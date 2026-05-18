package realtime

import (
	"bufio"
	"crypto/sha1"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"sync"
)

const (
	webSocketGUID   = "258EAFA5-E914-47DA-95CA-C5AB0DC85B11"
	maxFramePayload = 1 << 20
	opcodeText      = 0x1
	opcodeClose     = 0x8
	opcodePing      = 0x9
	opcodePong      = 0xA
)

var (
	errUnsupportedFrame = errors.New("unsupported websocket frame")
	errFrameTooLarge    = errors.New("websocket frame is too large")
	errUnmaskedFrame    = errors.New("client websocket frame is not masked")
)

type webSocketConn struct {
	conn   net.Conn
	reader *bufio.Reader
	writer *bufio.Writer
	mu     sync.Mutex
}

func acceptWebSocket(w http.ResponseWriter, r *http.Request) (*webSocketConn, error) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return nil, errors.New("websocket requires GET")
	}
	if !headerContains(r.Header, "Connection", "upgrade") || !strings.EqualFold(r.Header.Get("Upgrade"), "websocket") {
		http.Error(w, "upgrade required", http.StatusBadRequest)
		return nil, errors.New("websocket upgrade headers are missing")
	}
	if r.Header.Get("Sec-WebSocket-Version") != "13" {
		http.Error(w, "unsupported websocket version", http.StatusBadRequest)
		return nil, errors.New("unsupported websocket version")
	}

	key := r.Header.Get("Sec-WebSocket-Key")
	if key == "" {
		http.Error(w, "missing websocket key", http.StatusBadRequest)
		return nil, errors.New("missing websocket key")
	}

	hijacker, ok := w.(http.Hijacker)
	if !ok {
		http.Error(w, "websocket hijacking is not supported", http.StatusInternalServerError)
		return nil, errors.New("response writer does not support hijacking")
	}

	conn, rw, err := hijacker.Hijack()
	if err != nil {
		return nil, err
	}

	accept := webSocketAccept(key)
	_, err = fmt.Fprintf(rw, "HTTP/1.1 101 Switching Protocols\r\nUpgrade: websocket\r\nConnection: Upgrade\r\nSec-WebSocket-Accept: %s\r\n\r\n", accept)
	if err != nil {
		_ = conn.Close()
		return nil, err
	}
	if err := rw.Flush(); err != nil {
		_ = conn.Close()
		return nil, err
	}

	return &webSocketConn{
		conn:   conn,
		reader: rw.Reader,
		writer: rw.Writer,
	}, nil
}

func webSocketAccept(key string) string {
	hash := sha1.Sum([]byte(key + webSocketGUID))
	return base64.StdEncoding.EncodeToString(hash[:])
}

func headerContains(header http.Header, key string, value string) bool {
	for _, part := range strings.Split(header.Get(key), ",") {
		if strings.EqualFold(strings.TrimSpace(part), value) {
			return true
		}
	}

	return false
}

func (c *webSocketConn) ReadText() ([]byte, error) {
	for {
		opcode, payload, err := c.readFrame()
		if err != nil {
			return nil, err
		}

		switch opcode {
		case opcodeText:
			return payload, nil
		case opcodeClose:
			return nil, io.EOF
		case opcodePing:
			_ = c.writeFrame(opcodePong, payload)
		case opcodePong:
			continue
		default:
			return nil, errUnsupportedFrame
		}
	}
}

func (c *webSocketConn) WriteJSON(data []byte) error {
	return c.writeFrame(opcodeText, data)
}

func (c *webSocketConn) Close() error {
	return c.conn.Close()
}

func (c *webSocketConn) readFrame() (byte, []byte, error) {
	header := make([]byte, 2)
	if _, err := io.ReadFull(c.reader, header); err != nil {
		return 0, nil, err
	}

	fin := header[0]&0x80 != 0
	opcode := header[0] & 0x0F
	masked := header[1]&0x80 != 0
	payloadLen := uint64(header[1] & 0x7F)

	if !fin {
		return 0, nil, errUnsupportedFrame
	}
	if !masked {
		return 0, nil, errUnmaskedFrame
	}

	switch payloadLen {
	case 126:
		extended := make([]byte, 2)
		if _, err := io.ReadFull(c.reader, extended); err != nil {
			return 0, nil, err
		}
		payloadLen = uint64(binary.BigEndian.Uint16(extended))
	case 127:
		extended := make([]byte, 8)
		if _, err := io.ReadFull(c.reader, extended); err != nil {
			return 0, nil, err
		}
		payloadLen = binary.BigEndian.Uint64(extended)
	}

	if payloadLen > maxFramePayload {
		return 0, nil, errFrameTooLarge
	}

	maskKey := make([]byte, 4)
	if _, err := io.ReadFull(c.reader, maskKey); err != nil {
		return 0, nil, err
	}

	payload := make([]byte, payloadLen)
	if _, err := io.ReadFull(c.reader, payload); err != nil {
		return 0, nil, err
	}
	for i := range payload {
		payload[i] ^= maskKey[i%4]
	}

	return opcode, payload, nil
}

func (c *webSocketConn) writeFrame(opcode byte, payload []byte) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	header := []byte{0x80 | opcode}
	switch {
	case len(payload) < 126:
		header = append(header, byte(len(payload)))
	case len(payload) <= 0xFFFF:
		header = append(header, 126, 0, 0)
		binary.BigEndian.PutUint16(header[len(header)-2:], uint16(len(payload)))
	default:
		header = append(header, 127, 0, 0, 0, 0, 0, 0, 0, 0)
		binary.BigEndian.PutUint64(header[len(header)-8:], uint64(len(payload)))
	}

	if _, err := c.writer.Write(header); err != nil {
		return err
	}
	if _, err := c.writer.Write(payload); err != nil {
		return err
	}

	return c.writer.Flush()
}
