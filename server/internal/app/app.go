package app

import (
	"fmt"
	"log"
	"net/http"

	"bidking/server/internal/application"
	"bidking/server/internal/config"
	"bidking/server/internal/game"
	"bidking/server/internal/httpapi"
)

func Run() error {
	cfg := config.Load()
	gameplay, err := config.LoadGameplayData(cfg.SharedConfigDir)
	if err != nil {
		return err
	}

	roomService, err := application.NewRoomService(roomRules(gameplay.Rules), application.NewSequentialIDGenerator(), application.NewStaticLotProvider(lotsFromCollectibles(gameplay.Collectibles)))
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

func roomRules(data config.GameRulesData) game.RoomRules {
	rules := game.DefaultRoomRules()
	rules.MinPlayers = data.Players.Min
	rules.MaxPlayers = data.Players.Max
	rules.InitialGold = data.Players.InitialGold
	rules.RoundCount = data.Rounds.Count
	rules.RoundTimeSeconds = data.Rounds.TimeLimitSeconds
	rules.MinBid = data.Bidding.MinBid
	rules.MaxRebidRounds = data.Tiebreaker.MaxRounds
	return rules
}

func lotsFromCollectibles(collectibles []config.CollectibleData) []game.Lot {
	const lotSize = 5

	lots := make([]game.Lot, 0, (len(collectibles)+lotSize-1)/lotSize)
	for start := 0; start < len(collectibles); start += lotSize {
		end := start + lotSize
		if end > len(collectibles) {
			end = len(collectibles)
		}

		items := make([]game.Item, 0, end-start)
		trueValue := 0
		for _, collectible := range collectibles[start:end] {
			trueValue += collectible.TrueValue
			items = append(items, game.Item{
				ID:          collectible.ID,
				DisplayName: collectible.Name,
				Rarity:      collectible.Tier,
				Type:        collectible.Type,
				TrueValue:   collectible.TrueValue,
				SellValue:   collectible.SellValue,
			})
		}

		lots = append(lots, game.Lot{
			ID:          lotID(len(lots) + 1),
			DisplayName: "配置仓库",
			TrueValue:   trueValue,
			Items:       items,
		})
	}

	return lots
}

func lotID(index int) string {
	return fmt.Sprintf("lot_config_%03d", index)
}
