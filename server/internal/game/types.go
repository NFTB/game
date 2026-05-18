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
	ID          string   `json:"playerId"`
	DisplayName string   `json:"displayName"`
	Coins       int      `json:"coins"`
	Ready       bool     `json:"ready"`
	WonLotIDs   []string `json:"wonLotIds"`
}

type Item struct {
	ID                string `json:"itemId"`
	DisplayName       string `json:"displayName"`
	Rarity            string `json:"rarity"`
	Type              string `json:"type"`
	EstimatedMinValue int    `json:"estimatedMinValue"`
	EstimatedMaxValue int    `json:"estimatedMaxValue"`
	TrueValue         int    `json:"trueValue,omitempty"`
	SellValue         int    `json:"sellValue"`
}

type Lot struct {
	ID          string `json:"lotId"`
	DisplayName string `json:"displayName"`
	TrueValue   int    `json:"trueValue,omitempty"`
	Items       []Item `json:"items,omitempty"`
}

type Bid struct {
	PlayerID string `json:"playerId"`
	Amount   int    `json:"amount"`
}

type PlayerSnapshot struct {
	ID          string   `json:"playerId"`
	DisplayName string   `json:"displayName"`
	Coins       int      `json:"coins"`
	Ready       bool     `json:"ready"`
	WonLotIDs   []string `json:"wonLotIds"`
}

type BidSnapshot struct {
	PlayerID string `json:"playerId"`
	HasBid   bool   `json:"hasBid"`
}

type RoomSnapshot struct {
	RoomID           string           `json:"roomId"`
	Phase            RoomPhase        `json:"phase"`
	RoundNumber      int              `json:"roundNumber"`
	RoundTimeSeconds int              `json:"roundTimeSeconds"`
	Players          []PlayerSnapshot `json:"players"`
	CurrentLot       *Lot             `json:"currentLot,omitempty"`
	Bids             []BidSnapshot    `json:"bids"`
	RebidPlayerIDs   []string         `json:"rebidPlayerIds,omitempty"`
}

type RoundResult struct {
	RoundNumber   int          `json:"roundNumber"`
	Outcome       RoundOutcome `json:"outcome"`
	WinnerID      string       `json:"winnerId,omitempty"`
	WinningBid    int          `json:"winningBid,omitempty"`
	Lot           Lot          `json:"lot"`
	TiedPlayerIDs []string     `json:"tiedPlayerIds,omitempty"`
}
