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

	if result, err := service.PlaceBid(ctx, roomID, alice.PlayerID, 100); err != nil || !result.Accepted {
		t.Fatalf("alice bid result = %+v, %v; want accepted", result, err)
	}
	if result, err := service.PlaceBid(ctx, roomID, bob.PlayerID, 200); err != nil || !result.Accepted {
		t.Fatalf("bob bid result = %+v, %v; want accepted", result, err)
	}

	settled, err := service.SettleRound(ctx, roomID, alice.PlayerID)
	if err != nil {
		t.Fatalf("settle round: %v", err)
	}
	if settled.Result.WinnerID != bob.PlayerID {
		t.Fatalf("winner = %s, want %s", settled.Result.WinnerID, bob.PlayerID)
	}
	if settled.Snapshot.Phase != game.RoomPhaseSettlement {
		t.Fatalf("phase = %s, want %s", settled.Snapshot.Phase, game.RoomPhaseSettlement)
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
	if err := service.LeaveRoom(ctx, bob.PlayerID); err != nil {
		t.Fatalf("leave bob: %v", err)
	}
	if _, err := service.RoomIDForPlayer(ctx, bob.PlayerID); !errors.Is(err, ErrPlayerHasNoRoom) {
		t.Fatalf("room id for bob error = %v, want %v", err, ErrPlayerHasNoRoom)
	}

	snapshot, err := service.Snapshot(ctx, created.RoomID, alice.PlayerID)
	if err != nil {
		t.Fatalf("snapshot: %v", err)
	}
	if got := len(snapshot.Players); got != 1 {
		t.Fatalf("player count = %d, want 1", got)
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
