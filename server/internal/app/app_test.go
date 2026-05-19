package app

import (
	"testing"

	"bidking/server/internal/config"
)

func TestRoomRulesUseVenueEntryFee(t *testing.T) {
	data := config.GameplayData{}
	data.Rules.Players.Min = 2
	data.Rules.Players.Max = 4
	data.Rules.Players.InitialGold = 1000
	data.Rules.Rounds.Count = 5
	data.Rules.Rounds.TimeLimitSeconds = 30
	data.Rules.Bidding.MinBid = 1
	data.Rules.Tiebreaker.MaxRounds = 10
	data.Venues = []config.VenueData{{RoundEntryFee: 10}}

	rules := roomRules(data)
	if rules.RoundEntryFee != 10 {
		t.Fatalf("entry fee = %d, want 10", rules.RoundEntryFee)
	}
}

func TestLotsFromGameplayUseVenueCollectibleCount(t *testing.T) {
	data := config.GameplayData{}
	data.Rules.Rounds.Count = 2
	data.Venues = []config.VenueData{{}}
	data.Venues[0].CollectibleCountRange.Min = 3
	data.Collectibles = []config.CollectibleData{
		{ID: "c_1", Name: "一", TrueValue: 10, SellValue: 10},
		{ID: "c_2", Name: "二", TrueValue: 20, SellValue: 20},
	}

	lots := lotsFromGameplay(data)
	if len(lots) != 2 {
		t.Fatalf("lot count = %d, want 2", len(lots))
	}
	if len(lots[0].Items) != 3 || len(lots[1].Items) != 3 {
		t.Fatalf("lot item counts = %d, %d; want 3, 3", len(lots[0].Items), len(lots[1].Items))
	}
}
