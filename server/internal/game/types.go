package game

import "errors"

type RoomPhase string

const (
	RoomPhaseLobby      RoomPhase = "lobby"
	RoomPhaseAuction    RoomPhase = "auction"
	RoomPhaseRebid      RoomPhase = "rebid"
	RoomPhaseSettlement RoomPhase = "settlement"
	RoomPhaseFinished   RoomPhase = "finished"
)

type RoundOutcome string

const (
	RoundOutcomeAwarded    RoundOutcome = "awarded"
	RoundOutcomeNeedsRebid RoundOutcome = "needs_rebid"
	RoundOutcomeVoid       RoundOutcome = "void"
)

var (
	ErrRoomIDRequired      = errors.New("room id is required")
	ErrInvalidRoomRules    = errors.New("invalid room rules")
	ErrInvalidPhase        = errors.New("invalid room phase")
	ErrRoomFull            = errors.New("room is full")
	ErrPlayerAlreadyInRoom = errors.New("player is already in room")
	ErrPlayerNotInRoom     = errors.New("player is not in room")
	ErrInvalidPlayer       = errors.New("invalid player")
	ErrNotEnoughPlayers    = errors.New("not enough players")
	ErrNotAllReady         = errors.New("not all players are ready")
	ErrInvalidLot          = errors.New("invalid auction lot")
	ErrNoActiveLot         = errors.New("no active auction lot")
	ErrBidTooLow           = errors.New("bid is too low")
	ErrBidExceedsCoins     = errors.New("bid exceeds player coins")
	ErrPlayerNotInRebid    = errors.New("player is not eligible for rebid")
	ErrRebidTooLow         = errors.New("rebid must exceed previous bid")
	ErrMatchFinished       = errors.New("match is finished")
)

type RoomRules struct {
	MinPlayers       int
	MaxPlayers       int
	RoundCount       int
	MinBid           int
	MaxRebidRounds   int
	InitialGold      int
	RoundTimeSeconds int
}

func DefaultRoomRules() RoomRules {
	return RoomRules{
		MinPlayers:       2,
		MaxPlayers:       4,
		RoundCount:       5,
		MinBid:           1,
		MaxRebidRounds:   10,
		InitialGold:      10000000,
		RoundTimeSeconds: 30,
	}
}

type Player struct {
	ID          string
	DisplayName string
	Coins       int
	Ready       bool
	WonLotIDs   []string
}

type Item struct {
	ID                string
	DisplayName       string
	Rarity            string
	Type              string
	EstimatedMinValue int
	EstimatedMaxValue int
	TrueValue         int
	SellValue         int
}

type Lot struct {
	ID          string
	DisplayName string
	TrueValue   int
	Items       []Item
}

type Bid struct {
	PlayerID string
	Amount   int
}

type PlayerSnapshot struct {
	ID          string
	DisplayName string
	Coins       int
	Ready       bool
	WonLotIDs   []string
}

type BidSnapshot struct {
	PlayerID string
	HasBid   bool
}

type RoomSnapshot struct {
	RoomID           string
	Phase            RoomPhase
	RoundNumber      int
	RoundTimeSeconds int
	Players          []PlayerSnapshot
	CurrentLot       *Lot
	Bids             []BidSnapshot
	RebidPlayerIDs   []string
}

type RoundResult struct {
	RoundNumber   int
	Outcome       RoundOutcome
	WinnerID      string
	WinningBid    int
	Lot           Lot
	TiedPlayerIDs []string
}
