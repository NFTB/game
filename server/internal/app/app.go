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

	roomService, err := application.NewRoomService(roomRules(gameplay), application.NewSequentialIDGenerator(), application.NewStaticLotProvider(lotsFromGameplay(gameplay)))
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

func roomRules(data config.GameplayData) game.RoomRules {
	rules := game.DefaultRoomRules()
	rules.MinPlayers = data.Rules.Players.Min
	rules.MaxPlayers = data.Rules.Players.Max
	rules.InitialGold = data.Rules.Players.InitialGold
	rules.RoundCount = data.Rules.Rounds.Count
	rules.RoundTimeSeconds = data.Rules.Rounds.TimeLimitSeconds
	rules.MinBid = data.Rules.Bidding.MinBid
	rules.MaxRebidRounds = data.Rules.Tiebreaker.MaxRounds
	rules.RoundEntryFee = data.Venues[0].RoundEntryFee
	return rules
}

func lotsFromGameplay(data config.GameplayData) []game.Lot {
	lotSize := data.Venues[0].CollectibleCountRange.Min
	if lotSize <= 0 {
		lotSize = len(data.Collectibles)
	}

	lots := make([]game.Lot, 0, data.Rules.Rounds.Count)
	collectibleIndex := 0
	for round := 0; round < data.Rules.Rounds.Count; round++ {
		items := make([]game.Item, 0, lotSize)
		trueValue := 0
		for i := 0; i < lotSize; i++ {
			collectible := data.Collectibles[collectibleIndex%len(data.Collectibles)]
			collectibleIndex++

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
