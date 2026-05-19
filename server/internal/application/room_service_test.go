package application

import (
	"context"
	"errors"
	"testing"

	"bidking/server/internal/game"
)

func TestRoomServiceCreatesRoomAndStartsWhenAllReady(t *testing.T) {
	service := newTestRoomService(t)
	ctx := context.Background()

	alice := service.RegisterGuest(ctx, "Alice")
	bob := service.RegisterGuest(ctx, "Bob")

	created, err := service.CreateRoom(ctx, alice)
	if err != nil {
		t.Fatalf("create room: %v", err)
	}
	if created.RoomID == "" {
		t.Fatal("created room id is empty")
	}

	snapshot, err := service.JoinRoom(ctx, created.RoomID, bob)
	if err != nil {
		t.Fatalf("join room: %v", err)
	}
	if got := len(snapshot.Players); got != 2 {
		t.Fatalf("player count = %d, want 2", got)
	}

	snapshot, err = service.SetReady(ctx, created.RoomID, alice.PlayerID, true)
	if err != nil {
		t.Fatalf("ready alice: %v", err)
	}
	if snapshot.Phase != game.RoomPhaseLobby {
		t.Fatalf("phase after one ready = %s, want %s", snapshot.Phase, game.RoomPhaseLobby)
	}

	snapshot, err = service.SetReady(ctx, created.RoomID, bob.PlayerID, true)
	if err != nil {
		t.Fatalf("ready bob: %v", err)
	}
	if snapshot.Phase != game.RoomPhaseAuction {
		t.Fatalf("phase after all ready = %s, want %s", snapshot.Phase, game.RoomPhaseAuction)
	}
	if snapshot.CurrentLot == nil || snapshot.CurrentLot.ID != "lot_1" {
		t.Fatalf("current lot = %+v, want lot_1", snapshot.CurrentLot)
	}
}

func TestRoomServicePlacesBidAndSettlesRound(t *testing.T) {
	service := newTestRoomService(t)
	ctx := context.Background()
	roomID, alice, bob := readyStartedRoom(t, ctx, service)

	first, err := service.PlaceBid(ctx, roomID, alice.PlayerID, 100)
	if err != nil || !first.Accepted {
		t.Fatalf("alice bid result = %+v, %v; want accepted", first, err)
	}
	if first.RoundResult != nil {
		t.Fatalf("first bid round result = %+v, want nil", first.RoundResult)
	}

	second, err := service.PlaceBid(ctx, roomID, bob.PlayerID, 200)
	if err != nil || !second.Accepted {
		t.Fatalf("bob bid result = %+v, %v; want accepted", second, err)
	}
	if second.RoundResult == nil {
		t.Fatal("second bid should settle the round")
	}
	if second.RoundResult.WinnerID != bob.PlayerID {
		t.Fatalf("winner = %s, want %s", second.RoundResult.WinnerID, bob.PlayerID)
	}
	if second.Snapshot.Phase != game.RoomPhaseSettlement {
		t.Fatalf("phase = %s, want %s", second.Snapshot.Phase, game.RoomPhaseSettlement)
	}
}

func TestRoomServiceAutoSettlesWhenAllPlayersAct(t *testing.T) {
	service := newTestRoomService(t)
	ctx := context.Background()
	roomID, alice, bob := readyStartedRoom(t, ctx, service)

	first, err := service.PlaceBid(ctx, roomID, alice.PlayerID, 100)
	if err != nil {
		t.Fatalf("alice bid: %v", err)
	}
	if first.RoundResult != nil {
		t.Fatalf("first bid round result = %+v, want nil", first.RoundResult)
	}

	second, err := service.PassBid(ctx, roomID, bob.PlayerID)
	if err != nil {
		t.Fatalf("bob pass: %v", err)
	}
	if second.RoundResult == nil {
		t.Fatal("second action should settle the round")
	}
	if second.RoundResult.WinnerID != alice.PlayerID {
		t.Fatalf("winner = %s, want %s", second.RoundResult.WinnerID, alice.PlayerID)
	}
	if second.Snapshot.Phase != game.RoomPhaseSettlement {
		t.Fatalf("phase = %s, want %s", second.Snapshot.Phase, game.RoomPhaseSettlement)
	}
}

func TestRoomServiceReturnsSnapshotWhenBidRejected(t *testing.T) {
	service := newTestRoomService(t)
	ctx := context.Background()
	roomID, alice, _ := readyStartedRoom(t, ctx, service)

	result, err := service.PlaceBid(ctx, roomID, alice.PlayerID, alice.Coins+1)
	if !errors.Is(err, game.ErrBidExceedsCoins) {
		t.Fatalf("bid error = %v, want %v", err, game.ErrBidExceedsCoins)
	}
	if result.Accepted {
		t.Fatal("bid should not be accepted")
	}
	if result.Snapshot.RoomID != roomID {
		t.Fatalf("snapshot room = %s, want %s", result.Snapshot.RoomID, roomID)
	}
}

func TestRoomServiceLeaveRoomRemovesPlayerMapping(t *testing.T) {
	service := newTestRoomService(t)
	ctx := context.Background()

	alice := service.RegisterGuest(ctx, "Alice")
	bob := service.RegisterGuest(ctx, "Bob")
	created, err := service.CreateRoom(ctx, alice)
	if err != nil {
		t.Fatalf("create room: %v", err)
	}
	if _, err := service.JoinRoom(ctx, created.RoomID, bob); err != nil {
		t.Fatalf("join bob: %v", err)
	}
	snapshot, err := service.LeaveRoom(ctx, bob.PlayerID)
	if err != nil {
		t.Fatalf("leave bob: %v", err)
	}
	if got := len(snapshot.Players); got != 1 {
		t.Fatalf("leave snapshot player count = %d, want 1", got)
	}
	if _, err := service.RoomIDForPlayer(ctx, bob.PlayerID); !errors.Is(err, ErrPlayerHasNoRoom) {
		t.Fatalf("room id for bob error = %v, want %v", err, ErrPlayerHasNoRoom)
	}

	snapshot, err = service.Snapshot(ctx, created.RoomID, alice.PlayerID)
	if err != nil {
		t.Fatalf("snapshot: %v", err)
	}
	if got := len(snapshot.Players); got != 1 {
		t.Fatalf("player count = %d, want 1", got)
	}
}

func TestRoomServiceDisconnectRemovesLobbyPlayer(t *testing.T) {
	service := newTestRoomService(t)
	ctx := context.Background()

	alice := service.RegisterGuest(ctx, "Alice")
	bob := service.RegisterGuest(ctx, "Bob")
	created, err := service.CreateRoom(ctx, alice)
	if err != nil {
		t.Fatalf("create room: %v", err)
	}
	if _, err := service.JoinRoom(ctx, created.RoomID, bob); err != nil {
		t.Fatalf("join bob: %v", err)
	}

	disconnected, err := service.DisconnectPlayer(ctx, bob.PlayerID)
	if err != nil {
		t.Fatalf("disconnect bob: %v", err)
	}
	if disconnected.RoomID != created.RoomID || disconnected.RoomClosed {
		t.Fatalf("disconnect result = %+v, want open room", disconnected)
	}
	if got := len(disconnected.Snapshot.Players); got != 1 {
		t.Fatalf("player count = %d, want 1", got)
	}
	if _, err := service.RoomIDForPlayer(ctx, bob.PlayerID); !errors.Is(err, ErrPlayerHasNoRoom) {
		t.Fatalf("room id for bob error = %v, want %v", err, ErrPlayerHasNoRoom)
	}
}

func TestRoomServiceDisconnectDuringAuctionCanSettleRemainingActions(t *testing.T) {
	service := newTestRoomService(t)
	ctx := context.Background()
	roomID, alice, bob := readyStartedRoom(t, ctx, service)

	if _, err := service.PlaceBid(ctx, roomID, alice.PlayerID, 100); err != nil {
		t.Fatalf("alice bid: %v", err)
	}

	disconnected, err := service.DisconnectPlayer(ctx, bob.PlayerID)
	if err != nil {
		t.Fatalf("disconnect bob: %v", err)
	}
	if disconnected.RoundResult == nil {
		t.Fatal("disconnecting last pending bidder should settle the round")
	}
	if disconnected.RoundResult.WinnerID != alice.PlayerID {
		t.Fatalf("winner = %s, want %s", disconnected.RoundResult.WinnerID, alice.PlayerID)
	}
}

func TestRoomServiceAdvanceAfterSettlementStartsNextRound(t *testing.T) {
	service := newTestRoomService(t)
	ctx := context.Background()
	roomID, alice, bob := readyStartedRoom(t, ctx, service)

	if _, err := service.PlaceBid(ctx, roomID, alice.PlayerID, 100); err != nil {
		t.Fatalf("alice bid: %v", err)
	}
	if _, err := service.PassBid(ctx, roomID, bob.PlayerID); err != nil {
		t.Fatalf("bob pass: %v", err)
	}

	advanced, err := service.AdvanceAfterSettlement(ctx, roomID, alice.PlayerID)
	if err != nil {
		t.Fatalf("alice confirms settlement: %v", err)
	}
	if !advanced.Waiting {
		t.Fatalf("first confirmation = %+v, want waiting", advanced)
	}
	if advanced.Snapshot.Phase != game.RoomPhaseSettlement || advanced.Snapshot.RoundNumber != 1 {
		t.Fatalf("snapshot after first confirmation = %+v, want settlement round 1", advanced.Snapshot)
	}

	advanced, err = service.AdvanceAfterSettlement(ctx, roomID, bob.PlayerID)
	if err != nil {
		t.Fatalf("bob confirms settlement: %v", err)
	}
	if advanced.Finished {
		t.Fatal("room should not be finished after first round")
	}
	if !advanced.Advanced || advanced.Snapshot.Phase != game.RoomPhaseAuction || advanced.Snapshot.RoundNumber != 2 {
		t.Fatalf("snapshot = %+v, want auction round 2", advanced.Snapshot)
	}
}

func TestRoomServiceAdvanceAfterFinalSettlementFinishesMatch(t *testing.T) {
	rules := game.DefaultRoomRules()
	rules.RoundCount = 1
	rules.InitialGold = 1000
	service, err := NewRoomService(rules, NewSequentialIDGenerator(), NewStaticLotProvider([]game.Lot{
		{ID: "lot_1", DisplayName: "测试仓库", TrueValue: 500},
	}))
	if err != nil {
		t.Fatalf("new room service: %v", err)
	}
	ctx := context.Background()
	roomID, alice, bob := readyStartedRoom(t, ctx, service)

	if _, err := service.PlaceBid(ctx, roomID, alice.PlayerID, 100); err != nil {
		t.Fatalf("alice bid: %v", err)
	}
	if _, err := service.PassBid(ctx, roomID, bob.PlayerID); err != nil {
		t.Fatalf("bob pass: %v", err)
	}

	advanced, err := service.AdvanceAfterSettlement(ctx, roomID, alice.PlayerID)
	if err != nil {
		t.Fatalf("alice confirms final settlement: %v", err)
	}
	if !advanced.Waiting {
		t.Fatalf("first final confirmation = %+v, want waiting", advanced)
	}

	advanced, err = service.AdvanceAfterSettlement(ctx, roomID, bob.PlayerID)
	if err != nil {
		t.Fatalf("bob confirms final settlement: %v", err)
	}
	if !advanced.Finished {
		t.Fatal("room should be finished after final round")
	}
	if advanced.Snapshot.Phase != game.RoomPhaseFinished {
		t.Fatalf("phase = %s, want %s", advanced.Snapshot.Phase, game.RoomPhaseFinished)
	}
}

func newTestRoomService(t *testing.T) *RoomService {
	t.Helper()

	rules := game.DefaultRoomRules()
	rules.InitialGold = 1000
	service, err := NewRoomService(rules, NewSequentialIDGenerator(), NewStaticLotProvider([]game.Lot{
		{
			ID:          "lot_1",
			DisplayName: "测试仓库",
			TrueValue:   500,
			Items: []game.Item{
				{ID: "item_1", DisplayName: "测试藏品", TrueValue: 500, SellValue: 500},
			},
		},
	}))
	if err != nil {
		t.Fatalf("new room service: %v", err)
	}

	return service
}

func readyStartedRoom(t *testing.T, ctx context.Context, service *RoomService) (string, GuestSession, GuestSession) {
	t.Helper()

	alice := service.RegisterGuest(ctx, "Alice")
	bob := service.RegisterGuest(ctx, "Bob")
	created, err := service.CreateRoom(ctx, alice)
	if err != nil {
		t.Fatalf("create room: %v", err)
	}
	if _, err := service.JoinRoom(ctx, created.RoomID, bob); err != nil {
		t.Fatalf("join room: %v", err)
	}
	if _, err := service.SetReady(ctx, created.RoomID, alice.PlayerID, true); err != nil {
		t.Fatalf("ready alice: %v", err)
	}
	if _, err := service.SetReady(ctx, created.RoomID, bob.PlayerID, true); err != nil {
		t.Fatalf("ready bob: %v", err)
	}

	return created.RoomID, alice, bob
}
