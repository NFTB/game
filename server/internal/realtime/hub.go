package realtime

import "net/http"

type Hub struct {
	router *MessageRouter
}

func NewHub(rooms RoomCommands) (*Hub, error) {
	router, err := NewMessageRouter(rooms)
	if err != nil {
		return nil, err
	}

	return &Hub{router: router}, nil
}

func (h *Hub) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "websocket transport is not implemented yet", http.StatusNotImplemented)
}
