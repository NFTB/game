package app

import (
	"log"
	"net/http"

	"bidking/server/internal/config"
	"bidking/server/internal/httpapi"
)

func Run() error {
	cfg := config.Load()
	router := httpapi.NewRouter()

	server := &http.Server{
		Addr:              cfg.HTTPAddress,
		Handler:           router,
		ReadHeaderTimeout: cfg.ReadHeaderTimeout,
	}

	log.Printf("bidking server listening on %s", cfg.HTTPAddress)
	return server.ListenAndServe()
}
