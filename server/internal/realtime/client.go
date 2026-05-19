package realtime

import (
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"sync"
)

type Client struct {
	hub     *Hub
	conn    *webSocketConn
	send    chan []byte
	session ClientSession

	roomID string
	closed bool
}

type Hub struct {
	router *MessageRouter

	mu          sync.Mutex
	clients     map[*Client]struct{}
	roomClients map[string]map[*Client]struct{}
}

func NewHub(rooms RoomCommands) (*Hub, error) {
	router, err := NewMessageRouter(rooms)
	if err != nil {
		return nil, err
	}

	return &Hub{
		router:      router,
		clients:     make(map[*Client]struct{}),
		roomClients: make(map[string]map[*Client]struct{}),
	}, nil
}

func (h *Hub) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := acceptWebSocket(w, r)
	if err != nil {
		return
	}

	client := &Client{
		hub:  h,
		conn: conn,
		send: make(chan []byte, 16),
	}

	h.register(client)
	go client.writeLoop()
	client.readLoop(r)
}

func (h *Hub) register(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.clients[client] = struct{}{}
}

func (h *Hub) unregister(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if client.closed {
		return
	}

	delete(h.clients, client)
	h.removeFromRoomLocked(client)
	client.closed = true
	close(client.send)
}

func (h *Hub) disconnect(client *Client, r *http.Request) {
	if client.session.PlayerID == "" {
		return
	}

	result, err := h.router.rooms.DisconnectPlayer(r.Context(), client.session.PlayerID)
	if err != nil {
		return
	}
	if result.RoomClosed || result.RoomID == "" {
		return
	}
	if result.RoundResult != nil {
		h.broadcastRoom(result.RoomID, outbound("", "auction.round_settled", *result.RoundResult))
	}
	h.broadcastRoom(result.RoomID, outbound("", "room.snapshot", result.Snapshot))
}

func (h *Hub) refreshRoom(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if client.roomID == client.session.RoomID {
		return
	}

	h.removeFromRoomLocked(client)
	if client.session.RoomID == "" {
		return
	}

	if h.roomClients[client.session.RoomID] == nil {
		h.roomClients[client.session.RoomID] = make(map[*Client]struct{})
	}
	h.roomClients[client.session.RoomID][client] = struct{}{}
	client.roomID = client.session.RoomID
}

func (h *Hub) removeFromRoomLocked(client *Client) {
	if client.roomID == "" {
		return
	}

	clients := h.roomClients[client.roomID]
	delete(clients, client)
	if len(clients) == 0 {
		delete(h.roomClients, client.roomID)
	}
	client.roomID = ""
}

func (h *Hub) dispatch(sender *Client, responses []OutboundEnvelope) {
	h.refreshRoom(sender)
	for _, response := range responses {
		if isRoomBroadcast(response.Type) && sender.session.RoomID != "" {
			h.broadcastRoom(sender.session.RoomID, response)
			continue
		}

		h.send(sender, response)
	}

	if sender.session.PendingRoomClear {
		sender.session.RoomID = ""
		sender.session.PendingRoomClear = false
		h.refreshRoom(sender)
	}
}

func isRoomBroadcast(messageType string) bool {
	switch messageType {
	case "room.snapshot", "auction.round_settled", "auction.finished":
		return true
	default:
		return false
	}
}

func (h *Hub) broadcastRoom(roomID string, response OutboundEnvelope) {
	h.mu.Lock()
	clients := make([]*Client, 0, len(h.roomClients[roomID]))
	for client := range h.roomClients[roomID] {
		clients = append(clients, client)
	}
	h.mu.Unlock()

	for _, client := range clients {
		h.send(client, response)
	}
}

func (h *Hub) send(client *Client, response OutboundEnvelope) {
	data, err := json.Marshal(response)
	if err != nil {
		log.Printf("marshal outbound websocket message: %v", err)
		return
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	if client.closed {
		return
	}

	select {
	case client.send <- data:
	default:
		log.Printf("dropping websocket message for slow client")
	}
}

func (c *Client) readLoop(r *http.Request) {
	defer func() {
		c.hub.unregister(c)
		c.hub.disconnect(c, r)
		_ = c.conn.Close()
	}()

	for {
		data, err := c.conn.ReadText()
		if err != nil {
			if !errors.Is(err, io.EOF) {
				log.Printf("read websocket message: %v", err)
			}
			return
		}

		responses, err := c.hub.router.Route(r.Context(), &c.session, data)
		if len(responses) > 0 {
			c.hub.dispatch(c, responses)
		}
		if err != nil {
			log.Printf("route websocket message: %v", err)
		}
	}
}

func (c *Client) writeLoop() {
	for data := range c.send {
		if err := c.conn.WriteJSON(data); err != nil {
			log.Printf("write websocket message: %v", err)
			return
		}
	}
}
