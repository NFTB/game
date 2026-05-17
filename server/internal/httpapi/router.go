package httpapi

import (
	"encoding/json"
	"net/http"

	"bidking/server/internal/realtime"
)

func NewRouter() http.Handler {
	mux := http.NewServeMux()
	hub := realtime.NewHub()

	mux.HandleFunc("GET /healthz", handleHealthz)
	mux.HandleFunc("GET /ws", hub.HandleWebSocket)

	return mux
}

func handleHealthz(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]string{
		"status": "ok",
	})
}
