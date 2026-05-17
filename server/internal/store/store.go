package store

import "context"

type PlayerStore interface {
	CreateGuestPlayer(ctx context.Context, displayName string) (string, error)
}

type MatchStore interface {
	RecordMatchResult(ctx context.Context, result MatchResult) error
}

type MatchResult struct {
	RoomID   string
	WinnerID string
}
