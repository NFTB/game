package realtime

import (
	"context"
	"encoding/json"
	"testing"

	"bidking/server/internal/application"
	"bidking/server/internal/game"
)

func TestMessageRouterAuthCreateReadyAndBid(t *testing.T) {
	router := newTestRouter(t)
	ctx := context.Background()

	alice := &ClientSession{}
	bob := &ClientSession{}

	authAlice := route(t, router, ctx, alice, "auth.guest", "auth1", map[string]any{"displayName": "Alice"})
	if authAlice[0].Type != "auth.accepted" || alice.PlayerID == "" {
		t.Fatalf("auth alice response = %+v session = %+v", authAlice, alice)
	}
	authBob := route(t, router, ctx, bob, "auth.guest", "auth2", map[string]any{"displayName": "Bob"})
	if authBob[0].Type != "auth.accepted" || bob.PlayerID == "" {
		t.Fatalf("auth bob response = %+v session = %+v", authBob, bob)
	}

	created := route(t, router, ctx, alice, "room.create", "create1", map[string]any{})
	if created[0].Type != "room.snapshot" || alice.RoomID == "" {
		t.Fatalf("create room response = %+v session = %+v", created, alice)
	}

	joined := route(t, router, ctx, bob, "room.join", "join1", map[string]any{"roomId": alice.RoomID})
	if joined[0].Type != "room.snapshot" || bob.RoomID != alice.RoomID {
		t.Fatalf("join room response = %+v session = %+v", joined, bob)
	}

	route(t, router, ctx, alice, "room.ready", "ready1", map[string]any{"ready": true})
	readyBob := route(t, router, ctx, bob, "room.ready", "ready2", map[string]any{"ready": true})
	snapshot, ok := readyBob[0].Payload.(game.RoomSnapshot)
	if !ok {
		t.Fatalf("ready payload type = %T, want game.RoomSnapshot", readyBob[0].Payload)
	}
	if snapshot.Phase != game.RoomPhaseAuction {
		t.Fatalf("phase = %s, want %s", snapshot.Phase, game.RoomPhaseAuction)
	}

	bid := route(t, router, ctx, alice, "auction.bid", "bid1", map[string]any{"amount": 100})
	if len(bid) != 1 || bid[0].Type != "auction.bid_accepted" {
		t.Fatalf("bid response = %+v", bid)
	}
}

func TestMessageRouterRejectsBidWithoutRoom(t *testing.T) {
	router := newTestRouter(t)
	ctx := context.Background()
	session := &ClientSession{}
	route(t, router, ctx, session, "auth.guest", "auth1", map[string]any{"displayName": "Alice"})

	responses, err := router.Route(ctx, session, mustEnvelope(t, "auction.bid", "bid1", map[string]any{"amount": 100}))
	if err != ErrRoomRequired {
		t.Fatalf("route error = %v, want %v", err, ErrRoomRequired)
	}
	if len(responses) != 1 || responses[0].Type != "error" {
		t.Fatalf("responses = %+v, want error", responses)
	}
}

func TestMessageRouterEmitsRoundSettledWhenAllPlayersAct(t *testing.T) {
	router := newTestRouter(t)
	ctx := context.Background()

	alice := &ClientSession{}
	bob := &ClientSession{}
	route(t, router, ctx, alice, "auth.guest", "auth_alice", map[string]any{"displayName": "Alice"})
	route(t, router, ctx, bob, "auth.guest", "auth_bob", map[string]any{"displayName": "Bob"})
	route(t, router, ctx, alice, "room.create", "create", map[string]any{})
	route(t, router, ctx, bob, "room.join", "join", map[string]any{"roomId": alice.RoomID})
	route(t, router, ctx, alice, "room.ready", "ready_alice", map[string]any{"ready": true})
	route(t, router, ctx, bob, "room.ready", "ready_bob", map[string]any{"ready": true})

	route(t, router, ctx, alice, "auction.bid", "bid", map[string]any{"amount": 100})
	responses := route(t, router, ctx, bob, "auction.pass", "pass", map[string]any{})

	if len(responses) != 3 {
		t.Fatalf("response count = %d, want 3: %+v", len(responses), responses)
	}
	if responses[0].Type != "auction.pass_accepted" || responses[1].Type != "auction.round_settled" || responses[2].Type != "room.snapshot" {
		t.Fatalf("responses = %+v, want pass accepted, round settled, snapshot", responses)
	}
}

func TestMessageRouterAdvancesNextRound(t *testing.T) {
	router := newTestRouter(t)
	ctx := context.Background()

	alice := &ClientSession{}
	bob := &ClientSession{}
	route(t, router, ctx, alice, "auth.guest", "auth_alice", map[string]any{"displayName": "Alice"})
	route(t, router, ctx, bob, "auth.guest", "auth_bob", map[string]any{"displayName": "Bob"})
	route(t, router, ctx, alice, "room.create", "create", map[string]any{})
	route(t, router, ctx, bob, "room.join", "join", map[string]any{"roomId": alice.RoomID})
	route(t, router, ctx, alice, "room.ready", "ready_alice", map[string]any{"ready": true})
	route(t, router, ctx, bob, "room.ready", "ready_bob", map[string]any{"ready": true})
	route(t, router, ctx, alice, "auction.bid", "bid", map[string]any{"amount": 100})
	route(t, router, ctx, bob, "auction.pass", "pass", map[string]any{})

	responses := route(t, router, ctx, alice, "auction.next_round", "next_alice", map[string]any{})
	if len(responses) != 1 || responses[0].Type != "room.snapshot" {
		t.Fatalf("responses = %+v, want room.snapshot", responses)
	}
	snapshot, ok := responses[0].Payload.(game.RoomSnapshot)
	if !ok {
		t.Fatalf("payload type = %T, want game.RoomSnapshot", responses[0].Payload)
	}
	if snapshot.Phase != game.RoomPhaseSettlement || snapshot.RoundNumber != 1 {
		t.Fatalf("snapshot = %+v, want settlement round 1", snapshot)
	}

	responses = route(t, router, ctx, bob, "auction.next_round", "next_bob", map[string]any{})
	if len(responses) != 1 || responses[0].Type != "room.snapshot" {
		t.Fatalf("responses = %+v, want room.snapshot", responses)
	}
	snapshot, ok = responses[0].Payload.(game.RoomSnapshot)
	if !ok {
		t.Fatalf("payload type = %T, want game.RoomSnapshot", responses[0].Payload)
	}
	if snapshot.Phase != game.RoomPhaseAuction || snapshot.RoundNumber != 2 {
		t.Fatalf("snapshot = %+v, want auction round 2", snapshot)
	}
}

func TestMessageRouterLeaveRoomReturnsSnapshotAndMarksSessionForClear(t *testing.T) {
	router := newTestRouter(t)
	ctx := context.Background()

	alice := &ClientSession{}
	bob := &ClientSession{}
	route(t, router, ctx, alice, "auth.guest", "auth_alice", map[string]any{"displayName": "Alice"})
	route(t, router, ctx, bob, "auth.guest", "auth_bob", map[string]any{"displayName": "Bob"})
	route(t, router, ctx, alice, "room.create", "create", map[string]any{})
	route(t, router, ctx, bob, "room.join", "join", map[string]any{"roomId": alice.RoomID})

	responses := route(t, router, ctx, bob, "room.leave", "leave", map[string]any{})
	if len(responses) != 2 || responses[0].Type != "room.left" || responses[1].Type != "room.snapshot" {
		t.Fatalf("responses = %+v, want room.left and room.snapshot", responses)
	}
	if !bob.PendingRoomClear {
		t.Fatal("session should be marked for room clear after dispatch")
	}
}

func TestMessageRouterLeaveDuringAuctionEmitsSettlement(t *testing.T) {
	router := newTestRouter(t)
	ctx := context.Background()

	alice := &ClientSession{}
	bob := &ClientSession{}
	route(t, router, ctx, alice, "auth.guest", "auth_alice", map[string]any{"displayName": "Alice"})
	route(t, router, ctx, bob, "auth.guest", "auth_bob", map[string]any{"displayName": "Bob"})
	route(t, router, ctx, alice, "room.create", "create", map[string]any{})
	route(t, router, ctx, bob, "room.join", "join", map[string]any{"roomId": alice.RoomID})
	route(t, router, ctx, alice, "room.ready", "ready_alice", map[string]any{"ready": true})
	route(t, router, ctx, bob, "room.ready", "ready_bob", map[string]any{"ready": true})
	route(t, router, ctx, alice, "auction.bid", "bid", map[string]any{"amount": 100})

	responses := route(t, router, ctx, bob, "room.leave", "leave", map[string]any{})
	if len(responses) != 3 {
		t.Fatalf("response count = %d, want 3: %+v", len(responses), responses)
	}
	if responses[0].Type != "room.left" || responses[1].Type != "auction.round_settled" || responses[2].Type != "room.snapshot" {
		t.Fatalf("responses = %+v, want room.left, round_settled, snapshot", responses)
	}
}

func TestRankingsSortByFinalAssetValue(t *testing.T) {
	snapshot := game.RoomSnapshot{
		Players: []game.PlayerSnapshot{
			{ID: "player_b", Coins: 20, CollectionValue: 40},
			{ID: "player_c", Coins: 50, CollectionValue: 50},
			{ID: "player_a", Coins: 10, CollectionValue: 90},
		},
	}

	got := rankings(snapshot)
	if len(got) != 3 {
		t.Fatalf("rankings count = %d, want 3", len(got))
	}
	if got[0]["playerId"] != "player_a" || got[1]["playerId"] != "player_c" || got[2]["playerId"] != "player_b" {
		t.Fatalf("rankings order = %+v, want player_a, player_c, player_b", got)
	}
	if got[0]["totalCollectionValue"] != 90 {
		t.Fatalf("top ranking collection value = %v, want 90", got[0]["totalCollectionValue"])
	}
}

func newTestRouter(t *testing.T) *MessageRouter {
	t.Helper()

	rules := game.DefaultRoomRules()
	rules.InitialGold = 1000
	service, err := application.NewRoomService(rules, application.NewSequentialIDGenerator(), application.NewStaticLotProvider([]game.Lot{
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

	router, err := NewMessageRouter(service)
	if err != nil {
		t.Fatalf("new message router: %v", err)
	}

	return router
}

func route(t *testing.T, router *MessageRouter, ctx context.Context, session *ClientSession, messageType string, requestID string, payload any) []OutboundEnvelope {
	t.Helper()

	responses, err := router.Route(ctx, session, mustEnvelope(t, messageType, requestID, payload))
	if err != nil {
		t.Fatalf("route %s: %v", messageType, err)
	}

	return responses
}

func mustEnvelope(t *testing.T, messageType string, requestID string, payload any) []byte {
	t.Helper()

	data, err := json.Marshal(map[string]any{
		"type":      messageType,
		"requestId": requestID,
		"payload":   payload,
	})
	if err != nil {
		t.Fatalf("marshal envelope: %v", err)
	}

	return data
}
