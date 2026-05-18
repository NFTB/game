package application

import (
	"context"
	"errors"

	"bidking/server/internal/game"
)

var (
	ErrRoomNotFound       = errors.New("room not found")
	ErrPlayerHasNoRoom    = errors.New("player has no room")
	ErrLotProviderMissing = errors.New("lot provider is missing")
)

type IDGenerator interface {
	NewID(prefix string) string
}

type LotProvider interface {
	NextLot(ctx context.Context, snapshot game.RoomSnapshot) (game.Lot, error)
}

type GuestSession struct {
	PlayerID    string
	DisplayName string
	Coins       int
}

type CreateRoomResult struct {
	RoomID   string
	Snapshot game.RoomSnapshot
}

type PlaceBidResult struct {
	Accepted bool
	Snapshot game.RoomSnapshot
}

type SettleRoundResult struct {
	Result   game.RoundResult
	Snapshot game.RoomSnapshot
}
