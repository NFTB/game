package app

import (
	"log"
	"net/http"

	"bidking/server/internal/application"
	"bidking/server/internal/config"
	"bidking/server/internal/game"
	"bidking/server/internal/httpapi"
)

func Run() error {
	cfg := config.Load()
	roomService, err := application.NewRoomService(game.DefaultRoomRules(), application.NewSequentialIDGenerator(), application.NewStaticLotProvider(defaultLots()))
	if err != nil {
		return err
	}

	router, err := httpapi.NewRouter(roomService)
	if err != nil {
		return err
	}

	server := &http.Server{
		Addr:              cfg.HTTPAddress,
		Handler:           router,
		ReadHeaderTimeout: cfg.ReadHeaderTimeout,
	}

	log.Printf("bidking server listening on %s", cfg.HTTPAddress)
	return server.ListenAndServe()
}

func defaultLots() []game.Lot {
	return []game.Lot{
		{
			ID:          "lot_demo_001",
			DisplayName: "地下拍卖所仓库",
			TrueValue:   500000,
			Items: []game.Item{
				{
					ID:          "collectible_demo_001",
					DisplayName: "明代青花瓷瓶",
					Rarity:      "R",
					Type:        "瓷器",
					TrueValue:   450000,
					SellValue:   450000,
				},
				{
					ID:          "collectible_demo_002",
					DisplayName: "民国银元",
					Rarity:      "N",
					Type:        "杂项",
					TrueValue:   50000,
					SellValue:   50000,
				},
			},
		},
	}
}
