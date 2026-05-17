package game

type Room struct {
	ID          string
	Phase       RoomPhase
	Players     map[string]*Player
	CurrentItem *Item
	Bids        []Bid
}

func NewRoom(id string) *Room {
	return &Room{
		ID:      id,
		Phase:   RoomPhaseLobby,
		Players: make(map[string]*Player),
		Bids:    make([]Bid, 0),
	}
}

func (r *Room) AddPlayer(player *Player) {
	r.Players[player.ID] = player
}

func (r *Room) PlaceBid(playerID string, amount int) {
	r.Bids = append(r.Bids, Bid{
		PlayerID: playerID,
		Amount:   amount,
	})
}
