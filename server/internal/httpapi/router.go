package httpapi

import (
	"encoding/json"
	"net/http"

	"bidking/server/internal/realtime"
)

func NewRouter(rooms realtime.RoomCommands) (http.Handler, error) {
	mux := http.NewServeMux()
	hub, err := realtime.NewHub(rooms)
	if err != nil {
		return nil, err
	}

	mux.HandleFunc("GET /healthz", handleHealthz)
	mux.HandleFunc("GET /ws", hub.HandleWebSocket)

	return mux, nil
}

func handleHealthz(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]string{
		"status": "ok",
	})
}
