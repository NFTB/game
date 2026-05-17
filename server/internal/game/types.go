package game

type RoomPhase string

const (
	RoomPhaseLobby      RoomPhase = "lobby"
	RoomPhaseAuction    RoomPhase = "auction"
	RoomPhaseSettlement RoomPhase = "settlement"
)

type Player struct {
	ID          string
	DisplayName string
	Coins       int
	Ready       bool
}

type Item struct {
	ID                string
	DisplayName       string
	Rarity            string
	EstimatedMinValue int
	EstimatedMaxValue int
}

type Bid struct {
	PlayerID string
	Amount   int
}
