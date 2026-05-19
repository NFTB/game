package game

import "sort"

type Room struct {
	id          string
	phase       RoomPhase
	rules       RoomRules
	players     map[string]Player
	playerOrder []string

	roundNumber int
	currentLot  *Lot
	bids        map[string]Bid
	results     []RoundResult

	rebidRound        int
	rebidParticipants map[string]struct{}
	rebidFloors       map[string]int
	settlementReady   map[string]struct{}
}

func NewRoom(id string) *Room {
	room, _ := NewRoomWithRules(id, DefaultRoomRules())
	return room
}

func NewRoomWithRules(id string, rules RoomRules) (*Room, error) {
	if id == "" {
		return nil, ErrRoomIDRequired
	}

	normalized, err := normalizeRules(rules)
	if err != nil {
		return nil, err
	}

	return &Room{
		id:      id,
		phase:   RoomPhaseLobby,
		rules:   normalized,
		players: make(map[string]Player),
		bids:    make(map[string]Bid),
		results: make([]RoundResult, 0, normalized.RoundCount),
	}, nil
}

func normalizeRules(rules RoomRules) (RoomRules, error) {
	defaults := DefaultRoomRules()
	if rules == (RoomRules{}) {
		rules = defaults
	}
	if rules.MinPlayers == 0 {
		rules.MinPlayers = defaults.MinPlayers
	}
	if rules.MaxPlayers == 0 {
		rules.MaxPlayers = defaults.MaxPlayers
	}
	if rules.RoundCount == 0 {
		rules.RoundCount = defaults.RoundCount
	}
	if rules.MinBid == 0 {
		rules.MinBid = defaults.MinBid
	}
	if rules.InitialGold == 0 {
		rules.InitialGold = defaults.InitialGold
	}
	if rules.RoundTimeSeconds == 0 {
		rules.RoundTimeSeconds = defaults.RoundTimeSeconds
	}

	if rules.MinPlayers < 1 || rules.MaxPlayers < rules.MinPlayers || rules.RoundCount < 1 || rules.MinBid < 1 || rules.MaxRebidRounds < 0 || rules.InitialGold < 0 || rules.RoundTimeSeconds < 1 || rules.RoundEntryFee < 0 {
		return RoomRules{}, ErrInvalidRoomRules
	}

	return rules, nil
}

func (r *Room) ID() string {
	return r.id
}

func (r *Room) Phase() RoomPhase {
	return r.phase
}

func (r *Room) RoundNumber() int {
	return r.roundNumber
}

func (r *Room) Rules() RoomRules {
	return r.rules
}

func (r *Room) AddPlayer(player *Player) error {
	if player == nil {
		return ErrInvalidPlayer
	}

	return r.Join(*player)
}

func (r *Room) Join(player Player) error {
	if r.phase != RoomPhaseLobby {
		return ErrInvalidPhase
	}
	if player.ID == "" || player.Coins < 0 {
		return ErrInvalidPlayer
	}
	if _, exists := r.players[player.ID]; exists {
		return ErrPlayerAlreadyInRoom
	}
	if len(r.players) >= r.rules.MaxPlayers {
		return ErrRoomFull
	}

	player.Ready = false
	player.WonLotIDs = append([]string(nil), player.WonLotIDs...)
	r.players[player.ID] = player
	r.playerOrder = append(r.playerOrder, player.ID)
	return nil
}

func (r *Room) Leave(playerID string) error {
	if _, ok := r.players[playerID]; !ok {
		return ErrPlayerNotInRoom
	}

	delete(r.players, playerID)
	delete(r.bids, playerID)
	delete(r.rebidParticipants, playerID)
	delete(r.rebidFloors, playerID)
	delete(r.settlementReady, playerID)
	for i, orderedPlayerID := range r.playerOrder {
		if orderedPlayerID == playerID {
			r.playerOrder = append(r.playerOrder[:i], r.playerOrder[i+1:]...)
			break
		}
	}

	return nil
}

func (r *Room) PlayerCount() int {
	return len(r.players)
}

func (r *Room) SetReady(playerID string, ready bool) error {
	if r.phase != RoomPhaseLobby {
		return ErrInvalidPhase
	}

	player, ok := r.players[playerID]
	if !ok {
		return ErrPlayerNotInRoom
	}

	player.Ready = ready
	r.players[playerID] = player
	return nil
}

func (r *Room) AllReady() bool {
	if len(r.players) < r.rules.MinPlayers {
		return false
	}

	for _, player := range r.players {
		if !player.Ready {
			return false
		}
	}

	return true
}

func (r *Room) StartNextRound(lot Lot) error {
	switch r.phase {
	case RoomPhaseLobby:
		if !r.AllReady() {
			if len(r.players) < r.rules.MinPlayers {
				return ErrNotEnoughPlayers
			}
			return ErrNotAllReady
		}
	case RoomPhaseSettlement:
		if r.roundNumber >= r.rules.RoundCount {
			r.phase = RoomPhaseFinished
			return ErrMatchFinished
		}
	case RoomPhaseFinished:
		return ErrMatchFinished
	default:
		return ErrInvalidPhase
	}

	if lot.ID == "" {
		return ErrInvalidLot
	}
	if err := r.chargeRoundEntryFee(); err != nil {
		return err
	}

	r.roundNumber++
	r.currentLot = cloneLot(lot)
	r.bids = make(map[string]Bid)
	r.rebidRound = 0
	r.rebidParticipants = nil
	r.rebidFloors = nil
	r.settlementReady = nil
	r.phase = RoomPhaseAuction
	return nil
}

func (r *Room) chargeRoundEntryFee() error {
	if r.rules.RoundEntryFee == 0 {
		return nil
	}

	for _, player := range r.players {
		if player.Coins < r.rules.RoundEntryFee {
			return ErrEntryFeeExceedsCoins
		}
	}
	for playerID, player := range r.players {
		player.Coins -= r.rules.RoundEntryFee
		r.players[playerID] = player
	}

	return nil
}

func (r *Room) Finish() error {
	if r.phase != RoomPhaseSettlement {
		return ErrInvalidPhase
	}
	if r.roundNumber < r.rules.RoundCount {
		return ErrRoundsRemaining
	}

	r.phase = RoomPhaseFinished
	return nil
}

func (r *Room) ConfirmSettlement(playerID string) (bool, error) {
	if r.phase != RoomPhaseSettlement {
		return false, ErrInvalidPhase
	}
	if _, ok := r.players[playerID]; !ok {
		return false, ErrPlayerNotInRoom
	}
	if r.settlementReady == nil {
		r.settlementReady = make(map[string]struct{}, len(r.players))
	}

	r.settlementReady[playerID] = struct{}{}
	return len(r.settlementReady) == len(r.players), nil
}

func (r *Room) PlaceBid(playerID string, amount int) error {
	player, err := r.playerForAuctionAction(playerID)
	if err != nil {
		return err
	}
	if amount < r.rules.MinBid {
		return ErrBidTooLow
	}
	if amount > player.Coins {
		return ErrBidExceedsCoins
	}
	if r.phase == RoomPhaseRebid {
		if amount <= r.rebidFloors[playerID] {
			return ErrRebidTooLow
		}
	}

	r.bids[playerID] = Bid{
		PlayerID: playerID,
		Amount:   amount,
	}
	return nil
}

func (r *Room) Pass(playerID string) error {
	if _, err := r.playerForAuctionAction(playerID); err != nil {
		return err
	}

	r.bids[playerID] = Bid{
		PlayerID: playerID,
		Passed:   true,
	}
	return nil
}

func (r *Room) AllPlayersActed() bool {
	switch r.phase {
	case RoomPhaseAuction:
		return len(r.players) > 0 && len(r.bids) == len(r.players)
	case RoomPhaseRebid:
		return len(r.rebidParticipants) > 0 && len(r.bids) == len(r.rebidParticipants)
	default:
		return false
	}
}

func (r *Room) SettleRound() (RoundResult, error) {
	if r.phase != RoomPhaseAuction && r.phase != RoomPhaseRebid {
		return RoundResult{}, ErrInvalidPhase
	}
	if r.currentLot == nil {
		return RoundResult{}, ErrNoActiveLot
	}

	if len(r.bids) == 0 {
		return r.finishVoidRound(), nil
	}

	highest, tiedPlayerIDs := r.highestBidders()
	if len(tiedPlayerIDs) == 0 {
		return r.finishVoidRound(), nil
	}
	if len(tiedPlayerIDs) > 1 {
		if r.shouldVoidTiedRound() {
			return r.finishVoidRoundWithTies(tiedPlayerIDs), nil
		}

		r.enterRebid(tiedPlayerIDs)
		return RoundResult{
			RoundNumber:   r.roundNumber,
			Outcome:       RoundOutcomeNeedsRebid,
			Lot:           r.publicLot(),
			TiedPlayerIDs: append([]string(nil), tiedPlayerIDs...),
		}, nil
	}

	winnerID := tiedPlayerIDs[0]
	winner := r.players[winnerID]
	winner.Coins -= highest
	winner.WonLotIDs = append(winner.WonLotIDs, r.currentLot.ID)
	winner.CollectionValue += r.currentLot.TrueValue
	r.players[winnerID] = winner

	result := RoundResult{
		RoundNumber: r.roundNumber,
		Outcome:     RoundOutcomeAwarded,
		WinnerID:    winnerID,
		WinningBid:  highest,
		Lot:         *cloneLot(*r.currentLot),
	}
	r.results = append(r.results, result)
	r.phase = RoomPhaseSettlement
	r.bids = make(map[string]Bid)
	r.rebidParticipants = nil
	r.rebidFloors = nil
	r.settlementReady = make(map[string]struct{}, len(r.players))
	return result, nil
}

func (r *Room) SnapshotFor(_ string) RoomSnapshot {
	players := make([]PlayerSnapshot, 0, len(r.playerOrder))
	for _, playerID := range r.playerOrder {
		player := r.players[playerID]
		players = append(players, PlayerSnapshot{
			ID:              player.ID,
			DisplayName:     player.DisplayName,
			Coins:           player.Coins,
			Ready:           player.Ready,
			WonLotIDs:       append([]string(nil), player.WonLotIDs...),
			CollectionValue: player.CollectionValue,
		})
	}

	bids := make([]BidSnapshot, 0, len(r.bids))
	bidPlayerIDs := make([]string, 0, len(r.bids))
	for playerID := range r.bids {
		bidPlayerIDs = append(bidPlayerIDs, playerID)
	}
	sort.Strings(bidPlayerIDs)
	for _, playerID := range bidPlayerIDs {
		bids = append(bids, BidSnapshot{
			PlayerID: playerID,
			HasBid:   true,
		})
	}

	var lot *Lot
	if r.currentLot != nil {
		lot = r.snapshotLot()
	}

	return RoomSnapshot{
		RoomID:           r.id,
		Phase:            r.phase,
		RoundNumber:      r.roundNumber,
		RoundTimeSeconds: r.rules.RoundTimeSeconds,
		Players:          players,
		CurrentLot:       lot,
		Bids:             bids,
		RebidPlayerIDs:   r.rebidPlayerIDs(),
	}
}

func (r *Room) Results() []RoundResult {
	results := make([]RoundResult, 0, len(r.results))
	for _, result := range r.results {
		result.Lot = *cloneLot(result.Lot)
		result.TiedPlayerIDs = append([]string(nil), result.TiedPlayerIDs...)
		results = append(results, result)
	}

	return results
}

func (r *Room) snapshotLot() *Lot {
	if r.currentLot == nil {
		return nil
	}

	lot := cloneLot(*r.currentLot)
	if r.shouldRevealCurrentLot() {
		return lot
	}

	return &Lot{
		ID:          lot.ID,
		DisplayName: lot.DisplayName,
	}
}

func (r *Room) shouldRevealCurrentLot() bool {
	if r.phase != RoomPhaseSettlement && r.phase != RoomPhaseFinished {
		return false
	}
	if len(r.results) == 0 {
		return false
	}

	latest := r.results[len(r.results)-1]
	return latest.RoundNumber == r.roundNumber && latest.Outcome == RoundOutcomeAwarded
}

func (r *Room) publicLot() Lot {
	if r.currentLot == nil {
		return Lot{}
	}

	return Lot{
		ID:          r.currentLot.ID,
		DisplayName: r.currentLot.DisplayName,
	}
}

func (r *Room) highestBidders() (int, []string) {
	highest := 0
	tiedPlayerIDs := make([]string, 0)
	for _, bid := range r.bids {
		if bid.Passed {
			continue
		}
		if bid.Amount > highest {
			highest = bid.Amount
			tiedPlayerIDs = tiedPlayerIDs[:0]
			tiedPlayerIDs = append(tiedPlayerIDs, bid.PlayerID)
			continue
		}
		if bid.Amount == highest {
			tiedPlayerIDs = append(tiedPlayerIDs, bid.PlayerID)
		}
	}
	sort.Strings(tiedPlayerIDs)
	return highest, tiedPlayerIDs
}

func (r *Room) playerForAuctionAction(playerID string) (Player, error) {
	if r.phase != RoomPhaseAuction && r.phase != RoomPhaseRebid {
		return Player{}, ErrInvalidPhase
	}

	player, ok := r.players[playerID]
	if !ok {
		return Player{}, ErrPlayerNotInRoom
	}
	if r.phase == RoomPhaseRebid {
		if _, ok := r.rebidParticipants[playerID]; !ok {
			return Player{}, ErrPlayerNotInRebid
		}
	}

	return player, nil
}

func (r *Room) shouldVoidTiedRound() bool {
	return r.rules.MaxRebidRounds == 0 || (r.phase == RoomPhaseRebid && r.rebidRound >= r.rules.MaxRebidRounds)
}

func (r *Room) enterRebid(tiedPlayerIDs []string) {
	r.rebidRound++
	r.rebidParticipants = make(map[string]struct{}, len(tiedPlayerIDs))
	r.rebidFloors = make(map[string]int, len(tiedPlayerIDs))

	for _, playerID := range tiedPlayerIDs {
		r.rebidParticipants[playerID] = struct{}{}
		r.rebidFloors[playerID] = r.bids[playerID].Amount
	}

	r.bids = make(map[string]Bid)
	r.phase = RoomPhaseRebid
}

func (r *Room) finishVoidRound() RoundResult {
	result := RoundResult{
		RoundNumber: r.roundNumber,
		Outcome:     RoundOutcomeVoid,
		Lot:         r.publicLot(),
	}
	r.results = append(r.results, result)
	r.phase = RoomPhaseSettlement
	r.bids = make(map[string]Bid)
	r.rebidParticipants = nil
	r.rebidFloors = nil
	r.settlementReady = make(map[string]struct{}, len(r.players))
	return result
}

func (r *Room) finishVoidRoundWithTies(tiedPlayerIDs []string) RoundResult {
	result := RoundResult{
		RoundNumber:   r.roundNumber,
		Outcome:       RoundOutcomeVoid,
		Lot:           r.publicLot(),
		TiedPlayerIDs: append([]string(nil), tiedPlayerIDs...),
	}
	r.results = append(r.results, result)
	r.phase = RoomPhaseSettlement
	r.bids = make(map[string]Bid)
	r.rebidParticipants = nil
	r.rebidFloors = nil
	r.settlementReady = make(map[string]struct{}, len(r.players))
	return result
}

func (r *Room) rebidPlayerIDs() []string {
	if len(r.rebidParticipants) == 0 {
		return nil
	}

	playerIDs := make([]string, 0, len(r.rebidParticipants))
	for playerID := range r.rebidParticipants {
		playerIDs = append(playerIDs, playerID)
	}
	sort.Strings(playerIDs)
	return playerIDs
}

func cloneLot(lot Lot) *Lot {
	lot.Items = append([]Item(nil), lot.Items...)
	return &lot
}
