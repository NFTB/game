package realtime

import "net/http"

type Hub struct{}

func NewHub() *Hub {
	return &Hub{}
}

func (h *Hub) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "websocket transport is not implemented yet", http.StatusNotImplemented)
}
