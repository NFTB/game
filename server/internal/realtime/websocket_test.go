package realtime

import (
	"bufio"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"bidking/server/internal/application"
	"bidking/server/internal/game"
)

func TestHubWebSocketAuthRoundTrip(t *testing.T) {
	server := newWebSocketTestServer(t)
	defer server.Close()

	conn, reader := dialWebSocket(t, server.URL)
	defer conn.Close()

	writeClientTextFrame(t, conn, mustEnvelope(t, "auth.guest", "req_auth", map[string]any{"displayName": "Alice"}))
	message := readServerTextFrame(t, reader)

	var response struct {
		Type      string         `json:"type"`
		RequestID string         `json:"requestId"`
		Payload   map[string]any `json:"payload"`
	}
	if err := json.Unmarshal(message, &response); err != nil {
		t.Fatalf("unmarshal websocket response: %v", err)
	}
	if response.Type != "auth.accepted" {
		t.Fatalf("response type = %s, want auth.accepted", response.Type)
	}
	if response.RequestID != "req_auth" {
		t.Fatalf("request id = %s, want req_auth", response.RequestID)
	}
	if response.Payload["playerId"] == "" || response.Payload["displayName"] != "Alice" {
		t.Fatalf("unexpected auth payload: %+v", response.Payload)
	}
}

func TestHubBroadcastsRoomSnapshot(t *testing.T) {
	server := newWebSocketTestServer(t)
	defer server.Close()

	aliceConn, aliceReader := dialWebSocket(t, server.URL)
	defer aliceConn.Close()
	bobConn, bobReader := dialWebSocket(t, server.URL)
	defer bobConn.Close()

	writeClientTextFrame(t, aliceConn, mustEnvelope(t, "auth.guest", "auth_alice", map[string]any{"displayName": "Alice"}))
	readServerTextFrame(t, aliceReader)
	writeClientTextFrame(t, bobConn, mustEnvelope(t, "auth.guest", "auth_bob", map[string]any{"displayName": "Bob"}))
	readServerTextFrame(t, bobReader)

	writeClientTextFrame(t, aliceConn, mustEnvelope(t, "room.create", "create", map[string]any{}))
	aliceCreate := readTypedServerMessage(t, aliceReader)
	roomID := nestedString(t, aliceCreate, "payload", "roomId")

	writeClientTextFrame(t, bobConn, mustEnvelope(t, "room.join", "join", map[string]any{"roomId": roomID}))
	aliceBroadcast := readTypedServerMessage(t, aliceReader)
	bobJoin := readTypedServerMessage(t, bobReader)

	if aliceBroadcast["type"] != "room.snapshot" || bobJoin["type"] != "room.snapshot" {
		t.Fatalf("join responses = alice:%+v bob:%+v, want room.snapshot", aliceBroadcast, bobJoin)
	}
	if nestedString(t, aliceBroadcast, "payload", "roomId") != roomID {
		t.Fatalf("broadcast room id = %s, want %s", nestedString(t, aliceBroadcast, "payload", "roomId"), roomID)
	}
}

func newWebSocketTestServer(t *testing.T) *httptest.Server {
	t.Helper()

	rules := game.DefaultRoomRules()
	rules.InitialGold = 1000
	service, err := application.NewRoomService(rules, application.NewSequentialIDGenerator(), application.NewStaticLotProvider([]game.Lot{
		{ID: "lot_1", DisplayName: "测试仓库", TrueValue: 500},
	}))
	if err != nil {
		t.Fatalf("new room service: %v", err)
	}

	hub, err := NewHub(service)
	if err != nil {
		t.Fatalf("new hub: %v", err)
	}

	return httptest.NewServer(http.HandlerFunc(hub.HandleWebSocket))
}

func dialWebSocket(t *testing.T, serverURL string) (net.Conn, *bufio.Reader) {
	t.Helper()

	parsed, err := url.Parse(serverURL)
	if err != nil {
		t.Fatalf("parse server url: %v", err)
	}

	conn, err := net.Dial("tcp", parsed.Host)
	if err != nil {
		t.Fatalf("dial websocket server: %v", err)
	}
	if err := conn.SetDeadline(time.Now().Add(5 * time.Second)); err != nil {
		t.Fatalf("set websocket test deadline: %v", err)
	}
	t.Cleanup(func() {
		_ = conn.Close()
	})

	reader := bufio.NewReader(conn)
	key := "dGhlIHNhbXBsZSBub25jZQ=="
	request := fmt.Sprintf("GET / HTTP/1.1\r\nHost: %s\r\nUpgrade: websocket\r\nConnection: Upgrade\r\nSec-WebSocket-Version: 13\r\nSec-WebSocket-Key: %s\r\n\r\n", parsed.Host, key)
	if _, err := conn.Write([]byte(request)); err != nil {
		t.Fatalf("write websocket handshake: %v", err)
	}

	response, err := http.ReadResponse(reader, &http.Request{Method: http.MethodGet})
	if err != nil {
		t.Fatalf("read websocket handshake response: %v", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusSwitchingProtocols {
		t.Fatalf("handshake status = %d, want %d", response.StatusCode, http.StatusSwitchingProtocols)
	}
	if !strings.EqualFold(response.Header.Get("Upgrade"), "websocket") {
		t.Fatalf("upgrade response = %q, want websocket", response.Header.Get("Upgrade"))
	}
	if response.Header.Get("Sec-WebSocket-Accept") != webSocketAccept(key) {
		t.Fatalf("accept key = %s, want %s", response.Header.Get("Sec-WebSocket-Accept"), webSocketAccept(key))
	}

	return conn, reader
}

func writeClientTextFrame(t *testing.T, writer io.Writer, payload []byte) {
	t.Helper()

	header := []byte{0x81}
	switch {
	case len(payload) < 126:
		header = append(header, 0x80|byte(len(payload)))
	case len(payload) <= 0xFFFF:
		header = append(header, 0x80|126, 0, 0)
		binary.BigEndian.PutUint16(header[len(header)-2:], uint16(len(payload)))
	default:
		t.Fatalf("payload too large: %d", len(payload))
	}

	maskKey := []byte{1, 2, 3, 4}
	masked := append([]byte(nil), payload...)
	for i := range masked {
		masked[i] ^= maskKey[i%4]
	}

	frame := append(header, maskKey...)
	frame = append(frame, masked...)
	if _, err := writer.Write(frame); err != nil {
		t.Fatalf("write client websocket frame: %v", err)
	}
}

func readServerTextFrame(t *testing.T, reader *bufio.Reader) []byte {
	t.Helper()

	header := make([]byte, 2)
	if _, err := io.ReadFull(reader, header); err != nil {
		t.Fatalf("read server frame header: %v", err)
	}
	if header[0]&0x0F != opcodeText {
		t.Fatalf("server opcode = %d, want text", header[0]&0x0F)
	}

	payloadLen := uint64(header[1] & 0x7F)
	switch payloadLen {
	case 126:
		extended := make([]byte, 2)
		if _, err := io.ReadFull(reader, extended); err != nil {
			t.Fatalf("read extended payload length: %v", err)
		}
		payloadLen = uint64(binary.BigEndian.Uint16(extended))
	case 127:
		extended := make([]byte, 8)
		if _, err := io.ReadFull(reader, extended); err != nil {
			t.Fatalf("read extended payload length: %v", err)
		}
		payloadLen = binary.BigEndian.Uint64(extended)
	}

	payload := make([]byte, payloadLen)
	if _, err := io.ReadFull(reader, payload); err != nil {
		t.Fatalf("read server payload: %v", err)
	}

	return payload
}

func readTypedServerMessage(t *testing.T, reader *bufio.Reader) map[string]any {
	t.Helper()

	payload := readServerTextFrame(t, reader)
	var message map[string]any
	if err := json.Unmarshal(payload, &message); err != nil {
		t.Fatalf("unmarshal server message: %v", err)
	}

	return message
}

func nestedString(t *testing.T, message map[string]any, keys ...string) string {
	t.Helper()

	var current any = message
	for _, key := range keys {
		object, ok := current.(map[string]any)
		if !ok {
			t.Fatalf("value at %s is %T, want object", key, current)
		}
		current = object[key]
	}

	value, ok := current.(string)
	if !ok {
		t.Fatalf("value at %v is %T, want string", keys, current)
	}

	return value
}
