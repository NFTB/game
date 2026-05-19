package game

import (
	"errors"
	"testing"
)

func TestRoomJoinReadyAndStartRound(t *testing.T) {
	room := NewRoom("room_1")

	if err := room.Join(Player{ID: "player_1", DisplayName: "A", Coins: 100}); err != nil {
		t.Fatalf("join player 1: %v", err)
	}
	if err := room.Join(Player{ID: "player_2", DisplayName: "B", Coins: 100}); err != nil {
		t.Fatalf("join player 2: %v", err)
	}
	if err := room.Join(Player{ID: "player_2", DisplayName: "B", Coins: 100}); !errors.Is(err, ErrPlayerAlreadyInRoom) {
		t.Fatalf("duplicate join error = %v, want %v", err, ErrPlayerAlreadyInRoom)
	}
	if room.AllReady() {
		t.Fatal("room should not be ready before players set ready")
	}

	if err := room.SetReady("player_1", true); err != nil {
		t.Fatalf("ready player 1: %v", err)
	}
	if err := room.SetReady("player_2", true); err != nil {
		t.Fatalf("ready player 2: %v", err)
	}
	if !room.AllReady() {
		t.Fatal("room should be ready after all players set ready")
	}

	if err := room.StartNextRound(testLot("lot_1")); err != nil {
		t.Fatalf("start round: %v", err)
	}
	if got := room.Phase(); got != RoomPhaseAuction {
		t.Fatalf("phase = %s, want %s", got, RoomPhaseAuction)
	}
	if got := room.RoundNumber(); got != 1 {
		t.Fatalf("round number = %d, want 1", got)
	}

	snapshot := room.SnapshotFor("player_1")
	if snapshot.RoomID != "room_1" || len(snapshot.Players) != 2 || snapshot.CurrentLot == nil {
		t.Fatalf("unexpected snapshot: %+v", snapshot)
	}
	if snapshot.CurrentLot.TrueValue != 0 || len(snapshot.CurrentLot.Items) != 0 {
		t.Fatalf("auction snapshot leaked lot details: %+v", snapshot.CurrentLot)
	}
}

func TestNewRoomWithEmptyRulesUsesDefaults(t *testing.T) {
	room, err := NewRoomWithRules("room_1", RoomRules{})
	if err != nil {
		t.Fatalf("new room with empty rules: %v", err)
	}

	if room.Rules().MaxRebidRounds != DefaultRoomRules().MaxRebidRounds {
		t.Fatalf("max rebid rounds = %d, want %d", room.Rules().MaxRebidRounds, DefaultRoomRules().MaxRebidRounds)
	}
}

func TestStartNextRoundChargesEntryFee(t *testing.T) {
	rules := DefaultRoomRules()
	rules.RoundEntryFee = 10
	room := startedRoomWithPlayersAndRules(t, rules, []Player{
		{ID: "player_1", DisplayName: "A", Coins: 100},
		{ID: "player_2", DisplayName: "B", Coins: 100},
	})

	snapshot := room.SnapshotFor("player_1")
	if snapshot.Players[0].Coins != 90 || snapshot.Players[1].Coins != 90 {
		t.Fatalf("coins after entry fee = %d, %d; want 90, 90", snapshot.Players[0].Coins, snapshot.Players[1].Coins)
	}
}

func TestRoomLeaveRemovesPlayerBeforeRoundStarts(t *testing.T) {
	room := NewRoom("room_1")

	if err := room.Join(Player{ID: "player_1", DisplayName: "A", Coins: 100}); err != nil {
		t.Fatalf("join player 1: %v", err)
	}
	if err := room.Join(Player{ID: "player_2", DisplayName: "B", Coins: 100}); err != nil {
		t.Fatalf("join player 2: %v", err)
	}

	if err := room.Leave("player_1"); err != nil {
		t.Fatalf("leave player 1: %v", err)
	}

	snapshot := room.SnapshotFor("player_2")
	if got := len(snapshot.Players); got != 1 {
		t.Fatalf("player count after leave = %d, want 1", got)
	}
	if snapshot.Players[0].ID != "player_2" {
		t.Fatalf("remaining player = %s, want player_2", snapshot.Players[0].ID)
	}
	if err := room.SetReady("player_1", true); !errors.Is(err, ErrPlayerNotInRoom) {
		t.Fatalf("ready left player error = %v, want %v", err, ErrPlayerNotInRoom)
	}
}

func TestRoomLeaveDuringAuctionRemovesPendingAction(t *testing.T) {
	room := startedRoomWithPlayers(t, []Player{
		{ID: "player_1", DisplayName: "A", Coins: 100},
		{ID: "player_2", DisplayName: "B", Coins: 100},
		{ID: "player_3", DisplayName: "C", Coins: 100},
	})

	if err := room.PlaceBid("player_1", 10); err != nil {
		t.Fatalf("bid player 1: %v", err)
	}
	if err := room.PlaceBid("player_2", 20); err != nil {
		t.Fatalf("bid player 2: %v", err)
	}
	if err := room.Leave("player_3"); err != nil {
		t.Fatalf("leave player 3: %v", err)
	}
	if !room.AllPlayersActed() {
		t.Fatal("remaining players have all acted after player 3 leaves")
	}
}

func TestPlaceBidRejectsInvalidAmounts(t *testing.T) {
	room := startedRoom(t)

	if err := room.PlaceBid("player_1", 0); !errors.Is(err, ErrBidTooLow) {
		t.Fatalf("zero bid error = %v, want %v", err, ErrBidTooLow)
	}
	if err := room.PlaceBid("player_1", 101); !errors.Is(err, ErrBidExceedsCoins) {
		t.Fatalf("overspend bid error = %v, want %v", err, ErrBidExceedsCoins)
	}
	if err := room.PlaceBid("missing", 10); !errors.Is(err, ErrPlayerNotInRoom) {
		t.Fatalf("missing player error = %v, want %v", err, ErrPlayerNotInRoom)
	}
}

func TestSettleRoundAwardsHighestBidder(t *testing.T) {
	room := startedRoom(t)

	if err := room.PlaceBid("player_1", 30); err != nil {
		t.Fatalf("bid player 1: %v", err)
	}
	if err := room.PlaceBid("player_2", 50); err != nil {
		t.Fatalf("bid player 2: %v", err)
	}

	result, err := room.SettleRound()
	if err != nil {
		t.Fatalf("settle round: %v", err)
	}
	if result.Outcome != RoundOutcomeAwarded {
		t.Fatalf("outcome = %s, want %s", result.Outcome, RoundOutcomeAwarded)
	}
	if result.WinnerID != "player_2" || result.WinningBid != 50 {
		t.Fatalf("unexpected winner result: %+v", result)
	}
	if got := room.Phase(); got != RoomPhaseSettlement {
		t.Fatalf("phase = %s, want %s", got, RoomPhaseSettlement)
	}

	snapshot := room.SnapshotFor("player_2")
	player2 := snapshot.Players[1]
	if player2.Coins != 50 {
		t.Fatalf("winner coins = %d, want 50", player2.Coins)
	}
	if len(player2.WonLotIDs) != 1 || player2.WonLotIDs[0] != "lot_1" {
		t.Fatalf("winner lot ids = %v, want [lot_1]", player2.WonLotIDs)
	}
	if player2.CollectionValue != 120 {
		t.Fatalf("winner collection value = %d, want 120", player2.CollectionValue)
	}
	if snapshot.CurrentLot == nil || snapshot.CurrentLot.TrueValue != 120 || len(snapshot.CurrentLot.Items) != 1 {
		t.Fatalf("settlement snapshot should reveal lot details: %+v", snapshot.CurrentLot)
	}
}

func TestPassAllowsPlayerToSkipBid(t *testing.T) {
	room := startedRoom(t)

	if err := room.PlaceBid("player_1", 30); err != nil {
		t.Fatalf("bid player 1: %v", err)
	}
	if err := room.Pass("player_2"); err != nil {
		t.Fatalf("pass player 2: %v", err)
	}
	if !room.AllPlayersActed() {
		t.Fatal("all players should have acted after bid and pass")
	}

	result, err := room.SettleRound()
	if err != nil {
		t.Fatalf("settle round: %v", err)
	}
	if result.Outcome != RoundOutcomeAwarded || result.WinnerID != "player_1" {
		t.Fatalf("unexpected result: %+v", result)
	}
}

func TestAllPassesVoidRound(t *testing.T) {
	room := startedRoom(t)

	if err := room.Pass("player_1"); err != nil {
		t.Fatalf("pass player 1: %v", err)
	}
	if err := room.Pass("player_2"); err != nil {
		t.Fatalf("pass player 2: %v", err)
	}

	result, err := room.SettleRound()
	if err != nil {
		t.Fatalf("settle round: %v", err)
	}
	if result.Outcome != RoundOutcomeVoid {
		t.Fatalf("outcome = %s, want %s", result.Outcome, RoundOutcomeVoid)
	}
}

func TestTieEntersRebidAndThenAwardsWinner(t *testing.T) {
	room := startedRoomWithPlayers(t, []Player{
		{ID: "player_1", DisplayName: "A", Coins: 100},
		{ID: "player_2", DisplayName: "B", Coins: 100},
		{ID: "player_3", DisplayName: "C", Coins: 100},
	})

	if err := room.PlaceBid("player_1", 50); err != nil {
		t.Fatalf("bid player 1: %v", err)
	}
	if err := room.PlaceBid("player_2", 50); err != nil {
		t.Fatalf("bid player 2: %v", err)
	}
	if err := room.PlaceBid("player_3", 10); err != nil {
		t.Fatalf("bid player 3: %v", err)
	}

	result, err := room.SettleRound()
	if err != nil {
		t.Fatalf("settle tied round: %v", err)
	}
	if result.Outcome != RoundOutcomeNeedsRebid {
		t.Fatalf("outcome = %s, want %s", result.Outcome, RoundOutcomeNeedsRebid)
	}
	if got := room.Phase(); got != RoomPhaseRebid {
		t.Fatalf("phase = %s, want %s", got, RoomPhaseRebid)
	}
	if err := room.PlaceBid("player_3", 60); !errors.Is(err, ErrPlayerNotInRebid) {
		t.Fatalf("non tied rebid error = %v, want %v", err, ErrPlayerNotInRebid)
	}
	if err := room.PlaceBid("player_1", 50); !errors.Is(err, ErrRebidTooLow) {
		t.Fatalf("low rebid error = %v, want %v", err, ErrRebidTooLow)
	}

	if err := room.PlaceBid("player_1", 60); err != nil {
		t.Fatalf("rebid player 1: %v", err)
	}
	if err := room.PlaceBid("player_2", 70); err != nil {
		t.Fatalf("rebid player 2: %v", err)
	}

	result, err = room.SettleRound()
	if err != nil {
		t.Fatalf("settle rebid round: %v", err)
	}
	if result.Outcome != RoundOutcomeAwarded || result.WinnerID != "player_2" || result.WinningBid != 70 {
		t.Fatalf("unexpected rebid result: %+v", result)
	}
}

func TestTieVoidsImmediatelyWhenMaxRebidRoundsIsZero(t *testing.T) {
	rules := DefaultRoomRules()
	rules.MaxRebidRounds = 0
	room := startedRoomWithRules(t, rules)

	if err := room.PlaceBid("player_1", 10); err != nil {
		t.Fatalf("bid player 1: %v", err)
	}
	if err := room.PlaceBid("player_2", 10); err != nil {
		t.Fatalf("bid player 2: %v", err)
	}

	result, err := room.SettleRound()
	if err != nil {
		t.Fatalf("settle tied round: %v", err)
	}
	if result.Outcome != RoundOutcomeVoid {
		t.Fatalf("outcome = %s, want %s", result.Outcome, RoundOutcomeVoid)
	}
	if got := room.Phase(); got != RoomPhaseSettlement {
		t.Fatalf("phase = %s, want %s", got, RoomPhaseSettlement)
	}
}

func TestTieVoidsAfterMaxRebidRounds(t *testing.T) {
	rules := DefaultRoomRules()
	rules.MaxRebidRounds = 1
	room := startedRoomWithRules(t, rules)

	if err := room.PlaceBid("player_1", 10); err != nil {
		t.Fatalf("bid player 1: %v", err)
	}
	if err := room.PlaceBid("player_2", 10); err != nil {
		t.Fatalf("bid player 2: %v", err)
	}
	if result, err := room.SettleRound(); err != nil || result.Outcome != RoundOutcomeNeedsRebid {
		t.Fatalf("first settle = %+v, %v; want needs rebid", result, err)
	}

	if err := room.PlaceBid("player_1", 20); err != nil {
		t.Fatalf("rebid player 1: %v", err)
	}
	if err := room.PlaceBid("player_2", 20); err != nil {
		t.Fatalf("rebid player 2: %v", err)
	}

	result, err := room.SettleRound()
	if err != nil {
		t.Fatalf("settle final tie: %v", err)
	}
	if result.Outcome != RoundOutcomeVoid {
		t.Fatalf("outcome = %s, want %s", result.Outcome, RoundOutcomeVoid)
	}
	if got := room.Phase(); got != RoomPhaseSettlement {
		t.Fatalf("phase = %s, want %s", got, RoomPhaseSettlement)
	}

	snapshot := room.SnapshotFor("player_1")
	if snapshot.Players[0].Coins != 100 || snapshot.Players[1].Coins != 100 {
		t.Fatalf("coins after void = %d, %d; want 100, 100", snapshot.Players[0].Coins, snapshot.Players[1].Coins)
	}
}

func TestFinishMovesSettledFinalRoundToFinished(t *testing.T) {
	rules := DefaultRoomRules()
	rules.RoundCount = 1
	room := startedRoomWithRules(t, rules)

	if err := room.PlaceBid("player_1", 10); err != nil {
		t.Fatalf("bid player 1: %v", err)
	}
	if err := room.Pass("player_2"); err != nil {
		t.Fatalf("pass player 2: %v", err)
	}
	if _, err := room.SettleRound(); err != nil {
		t.Fatalf("settle round: %v", err)
	}

	if err := room.Finish(); err != nil {
		t.Fatalf("finish room: %v", err)
	}
	if got := room.Phase(); got != RoomPhaseFinished {
		t.Fatalf("phase = %s, want %s", got, RoomPhaseFinished)
	}
}

func TestConfirmSettlementRequiresAllPlayers(t *testing.T) {
	room := startedRoom(t)
	if err := room.PlaceBid("player_1", 10); err != nil {
		t.Fatalf("bid player 1: %v", err)
	}
	if err := room.Pass("player_2"); err != nil {
		t.Fatalf("pass player 2: %v", err)
	}
	if _, err := room.SettleRound(); err != nil {
		t.Fatalf("settle round: %v", err)
	}

	ready, err := room.ConfirmSettlement("player_1")
	if err != nil {
		t.Fatalf("confirm player 1: %v", err)
	}
	if ready {
		t.Fatal("settlement should wait for player 2")
	}

	ready, err = room.ConfirmSettlement("player_2")
	if err != nil {
		t.Fatalf("confirm player 2: %v", err)
	}
	if !ready {
		t.Fatal("settlement should be ready after all players confirm")
	}
}

func startedRoom(t *testing.T) *Room {
	t.Helper()
	return startedRoomWithRules(t, DefaultRoomRules())
}

func startedRoomWithRules(t *testing.T, rules RoomRules) *Room {
	t.Helper()
	return startedRoomWithPlayersAndRules(t, rules, []Player{
		{ID: "player_1", DisplayName: "A", Coins: 100},
		{ID: "player_2", DisplayName: "B", Coins: 100},
	})
}

func startedRoomWithPlayers(t *testing.T, players []Player) *Room {
	t.Helper()
	return startedRoomWithPlayersAndRules(t, DefaultRoomRules(), players)
}

func startedRoomWithPlayersAndRules(t *testing.T, rules RoomRules, players []Player) *Room {
	t.Helper()

	room, err := NewRoomWithRules("room_1", rules)
	if err != nil {
		t.Fatalf("new room: %v", err)
	}
	for _, player := range players {
		if err := room.Join(player); err != nil {
			t.Fatalf("join %s: %v", player.ID, err)
		}
		if err := room.SetReady(player.ID, true); err != nil {
			t.Fatalf("ready %s: %v", player.ID, err)
		}
	}
	if err := room.StartNextRound(testLot("lot_1")); err != nil {
		t.Fatalf("start round: %v", err)
	}

	return room
}

func testLot(id string) Lot {
	return Lot{
		ID:          id,
		DisplayName: "测试仓库",
		TrueValue:   120,
		Items: []Item{
			{ID: "item_1", DisplayName: "测试藏品", TrueValue: 120, SellValue: 120},
		},
	}
}
