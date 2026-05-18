package realtime

import (
	"context"
	"encoding/json"
	"errors"

	"bidking/server/internal/application"
	"bidking/server/internal/game"
)

var (
	ErrRoomCommandsMissing = errors.New("room commands are missing")
	ErrUnauthenticated     = errors.New("client is unauthenticated")
	ErrRoomRequired        = errors.New("client is not in a room")
	ErrUnknownMessageType  = errors.New("unknown message type")
)

type RoomCommands interface {
	RegisterGuest(ctx context.Context, displayName string) application.GuestSession
	CreateRoom(ctx context.Context, guest application.GuestSession) (application.CreateRoomResult, error)
	JoinRoom(ctx context.Context, roomID string, guest application.GuestSession) (game.RoomSnapshot, error)
	LeaveRoom(ctx context.Context, playerID string) error
	SetReady(ctx context.Context, roomID string, playerID string, ready bool) (game.RoomSnapshot, error)
	PlaceBid(ctx context.Context, roomID string, playerID string, amount int) (application.PlaceBidResult, error)
	PassBid(ctx context.Context, roomID string, playerID string) (application.PlaceBidResult, error)
	SettleRound(ctx context.Context, roomID string, playerID string) (application.SettleRoundResult, error)
	RoomIDForPlayer(ctx context.Context, playerID string) (string, error)
}

type ClientSession struct {
	PlayerID    string
	DisplayName string
	Coins       int
	RoomID      string
}

type MessageRouter struct {
	rooms RoomCommands
}

func NewMessageRouter(rooms RoomCommands) (*MessageRouter, error) {
	if rooms == nil {
		return nil, ErrRoomCommandsMissing
	}

	return &MessageRouter{rooms: rooms}, nil
}

func (r *MessageRouter) Route(ctx context.Context, session *ClientSession, data []byte) ([]OutboundEnvelope, error) {
	var envelope Envelope
	if err := json.Unmarshal(data, &envelope); err != nil {
		return []OutboundEnvelope{outboundError("", "bad_request", "invalid message envelope")}, err
	}

	switch envelope.Type {
	case "auth.guest":
		return r.handleAuthGuest(ctx, session, envelope)
	case "room.create":
		return r.handleRoomCreate(ctx, session, envelope)
	case "room.join":
		return r.handleRoomJoin(ctx, session, envelope)
	case "room.leave":
		return r.handleRoomLeave(ctx, session, envelope)
	case "room.ready":
		return r.handleRoomReady(ctx, session, envelope)
	case "auction.bid":
		return r.handleAuctionBid(ctx, session, envelope)
	case "auction.pass":
		return r.handleAuctionPass(ctx, session, envelope)
	default:
		return []OutboundEnvelope{outboundError(envelope.RequestID, "unknown_message_type", "unknown message type")}, ErrUnknownMessageType
	}
}

func (r *MessageRouter) handleAuthGuest(ctx context.Context, session *ClientSession, envelope Envelope) ([]OutboundEnvelope, error) {
	var payload struct {
		DisplayName string `json:"displayName"`
	}
	if err := decodePayload(envelope.Payload, &payload); err != nil {
		return []OutboundEnvelope{outboundError(envelope.RequestID, "bad_payload", "invalid auth.guest payload")}, err
	}

	guest := r.rooms.RegisterGuest(ctx, payload.DisplayName)
	session.PlayerID = guest.PlayerID
	session.DisplayName = guest.DisplayName
	session.Coins = guest.Coins

	return []OutboundEnvelope{
		outbound(envelope.RequestID, "auth.accepted", map[string]any{
			"playerId":    guest.PlayerID,
			"displayName": guest.DisplayName,
		}),
	}, nil
}

func (r *MessageRouter) handleRoomCreate(ctx context.Context, session *ClientSession, envelope Envelope) ([]OutboundEnvelope, error) {
	guest, err := r.requireGuest(session)
	if err != nil {
		return []OutboundEnvelope{outboundError(envelope.RequestID, "unauthenticated", "authenticate before creating a room")}, err
	}

	result, err := r.rooms.CreateRoom(ctx, guest)
	if err != nil {
		return []OutboundEnvelope{outboundError(envelope.RequestID, "room_create_failed", err.Error())}, err
	}

	session.RoomID = result.RoomID
	return []OutboundEnvelope{
		outbound(envelope.RequestID, "room.snapshot", result.Snapshot),
	}, nil
}

func (r *MessageRouter) handleRoomJoin(ctx context.Context, session *ClientSession, envelope Envelope) ([]OutboundEnvelope, error) {
	guest, err := r.requireGuest(session)
	if err != nil {
		return []OutboundEnvelope{outboundError(envelope.RequestID, "unauthenticated", "authenticate before joining a room")}, err
	}

	var payload struct {
		RoomID string `json:"roomId"`
	}
	if err := decodePayload(envelope.Payload, &payload); err != nil {
		return []OutboundEnvelope{outboundError(envelope.RequestID, "bad_payload", "invalid room.join payload")}, err
	}

	snapshot, err := r.rooms.JoinRoom(ctx, payload.RoomID, guest)
	if err != nil {
		return []OutboundEnvelope{outboundError(envelope.RequestID, "room_join_failed", err.Error())}, err
	}

	session.RoomID = payload.RoomID
	return []OutboundEnvelope{
		outbound(envelope.RequestID, "room.snapshot", snapshot),
	}, nil
}

func (r *MessageRouter) handleRoomLeave(ctx context.Context, session *ClientSession, envelope Envelope) ([]OutboundEnvelope, error) {
	if session.PlayerID == "" {
		return []OutboundEnvelope{outboundError(envelope.RequestID, "unauthenticated", "authenticate before leaving a room")}, ErrUnauthenticated
	}

	if err := r.rooms.LeaveRoom(ctx, session.PlayerID); err != nil {
		return []OutboundEnvelope{outboundError(envelope.RequestID, "room_leave_failed", err.Error())}, err
	}

	session.RoomID = ""
	return []OutboundEnvelope{
		outbound(envelope.RequestID, "room.left", map[string]any{}),
	}, nil
}

func (r *MessageRouter) handleRoomReady(ctx context.Context, session *ClientSession, envelope Envelope) ([]OutboundEnvelope, error) {
	if err := r.requireRoom(session); err != nil {
		return []OutboundEnvelope{outboundError(envelope.RequestID, "room_required", "join a room before setting ready")}, err
	}

	var payload struct {
		Ready bool `json:"ready"`
	}
	if err := decodePayload(envelope.Payload, &payload); err != nil {
		return []OutboundEnvelope{outboundError(envelope.RequestID, "bad_payload", "invalid room.ready payload")}, err
	}

	snapshot, err := r.rooms.SetReady(ctx, session.RoomID, session.PlayerID, payload.Ready)
	if err != nil {
		return []OutboundEnvelope{outboundError(envelope.RequestID, "room_ready_failed", err.Error())}, err
	}

	return []OutboundEnvelope{
		outbound(envelope.RequestID, "room.snapshot", snapshot),
	}, nil
}

func (r *MessageRouter) handleAuctionBid(ctx context.Context, session *ClientSession, envelope Envelope) ([]OutboundEnvelope, error) {
	if err := r.requireRoom(session); err != nil {
		return []OutboundEnvelope{outboundError(envelope.RequestID, "room_required", "join a room before bidding")}, err
	}

	var payload struct {
		Amount int `json:"amount"`
	}
	if err := decodePayload(envelope.Payload, &payload); err != nil {
		return []OutboundEnvelope{outboundError(envelope.RequestID, "bad_payload", "invalid auction.bid payload")}, err
	}

	result, err := r.rooms.PlaceBid(ctx, session.RoomID, session.PlayerID, payload.Amount)
	if err != nil {
		responses := []OutboundEnvelope{
			outbound(envelope.RequestID, "auction.bid_rejected", ErrorPayload{Code: "bid_rejected", Message: err.Error()}),
		}
		if result.Snapshot.RoomID != "" {
			responses = append(responses, outbound(envelope.RequestID, "room.snapshot", result.Snapshot))
		}

		return responses, nil
	}

	return r.auctionAcceptedResponses(envelope.RequestID, "auction.bid_accepted", session.PlayerID, result), nil
}

func (r *MessageRouter) handleAuctionPass(ctx context.Context, session *ClientSession, envelope Envelope) ([]OutboundEnvelope, error) {
	if err := r.requireRoom(session); err != nil {
		return []OutboundEnvelope{outboundError(envelope.RequestID, "room_required", "join a room before passing")}, err
	}

	result, err := r.rooms.PassBid(ctx, session.RoomID, session.PlayerID)
	if err != nil {
		responses := []OutboundEnvelope{
			outbound(envelope.RequestID, "auction.bid_rejected", ErrorPayload{Code: "pass_rejected", Message: err.Error()}),
		}
		if result.Snapshot.RoomID != "" {
			responses = append(responses, outbound(envelope.RequestID, "room.snapshot", result.Snapshot))
		}

		return responses, nil
	}

	return r.auctionAcceptedResponses(envelope.RequestID, "auction.pass_accepted", session.PlayerID, result), nil
}

func (r *MessageRouter) auctionAcceptedResponses(requestID string, acceptedType string, playerID string, result application.PlaceBidResult) []OutboundEnvelope {
	responses := []OutboundEnvelope{
		outbound(requestID, acceptedType, map[string]any{
			"playerId": playerID,
		}),
	}
	if result.RoundResult != nil {
		responses = append(responses, outbound(requestID, "auction.round_settled", *result.RoundResult))
	}
	responses = append(responses, outbound(requestID, "room.snapshot", result.Snapshot))

	return responses
}

func (r *MessageRouter) requireGuest(session *ClientSession) (application.GuestSession, error) {
	if session == nil || session.PlayerID == "" {
		return application.GuestSession{}, ErrUnauthenticated
	}

	return application.GuestSession{
		PlayerID:    session.PlayerID,
		DisplayName: session.DisplayName,
		Coins:       session.Coins,
	}, nil
}

func (r *MessageRouter) requireRoom(session *ClientSession) error {
	if session == nil || session.PlayerID == "" {
		return ErrUnauthenticated
	}
	if session.RoomID == "" {
		return ErrRoomRequired
	}

	return nil
}
