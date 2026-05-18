package application

import (
	"context"
	"sync"

	"bidking/server/internal/game"
)

type RoomService struct {
	mu sync.Mutex

	rules       game.RoomRules
	ids         IDGenerator
	lotProvider LotProvider

	rooms       map[string]*game.Room
	playerRooms map[string]string
}

func NewRoomService(rules game.RoomRules, ids IDGenerator, lotProvider LotProvider) (*RoomService, error) {
	if ids == nil {
		ids = NewSequentialIDGenerator()
	}
	if lotProvider == nil {
		return nil, ErrLotProviderMissing
	}

	probe, err := game.NewRoomWithRules("room_rules_probe", rules)
	if err != nil {
		return nil, err
	}

	return &RoomService{
		rules:       probe.Rules(),
		ids:         ids,
		lotProvider: lotProvider,
		rooms:       make(map[string]*game.Room),
		playerRooms: make(map[string]string),
	}, nil
}

func (s *RoomService) RegisterGuest(_ context.Context, displayName string) GuestSession {
	playerID := s.ids.NewID("player")
	if displayName == "" {
		displayName = playerID
	}

	return GuestSession{
		PlayerID:    playerID,
		DisplayName: displayName,
		Coins:       s.rules.InitialGold,
	}
}

func (s *RoomService) CreateRoom(_ context.Context, guest GuestSession) (CreateRoomResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.playerRooms[guest.PlayerID]; exists {
		return CreateRoomResult{}, game.ErrPlayerAlreadyInRoom
	}

	roomID := s.ids.NewID("room")
	room, err := game.NewRoomWithRules(roomID, s.rules)
	if err != nil {
		return CreateRoomResult{}, err
	}

	player := game.Player{
		ID:          guest.PlayerID,
		DisplayName: guest.DisplayName,
		Coins:       guest.Coins,
	}
	if err := room.Join(player); err != nil {
		return CreateRoomResult{}, err
	}

	s.rooms[roomID] = room
	s.playerRooms[guest.PlayerID] = roomID
	return CreateRoomResult{
		RoomID:   roomID,
		Snapshot: room.SnapshotFor(guest.PlayerID),
	}, nil
}

func (s *RoomService) JoinRoom(_ context.Context, roomID string, guest GuestSession) (game.RoomSnapshot, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.playerRooms[guest.PlayerID]; exists {
		return game.RoomSnapshot{}, game.ErrPlayerAlreadyInRoom
	}

	room, err := s.room(roomID)
	if err != nil {
		return game.RoomSnapshot{}, err
	}

	player := game.Player{
		ID:          guest.PlayerID,
		DisplayName: guest.DisplayName,
		Coins:       guest.Coins,
	}
	if err := room.Join(player); err != nil {
		return game.RoomSnapshot{}, err
	}

	s.playerRooms[guest.PlayerID] = roomID
	return room.SnapshotFor(guest.PlayerID), nil
}

func (s *RoomService) LeaveRoom(_ context.Context, playerID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	roomID, ok := s.playerRooms[playerID]
	if !ok {
		return ErrPlayerHasNoRoom
	}

	room, err := s.room(roomID)
	if err != nil {
		return err
	}
	if err := room.Leave(playerID); err != nil {
		return err
	}

	delete(s.playerRooms, playerID)
	return nil
}

func (s *RoomService) SetReady(ctx context.Context, roomID string, playerID string, ready bool) (game.RoomSnapshot, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	room, err := s.room(roomID)
	if err != nil {
		return game.RoomSnapshot{}, err
	}
	if err := room.SetReady(playerID, ready); err != nil {
		return game.RoomSnapshot{}, err
	}

	if ready && room.Phase() == game.RoomPhaseLobby && room.AllReady() {
		lot, err := s.lotProvider.NextLot(ctx, room.SnapshotFor(playerID))
		if err != nil {
			return game.RoomSnapshot{}, err
		}
		if err := room.StartNextRound(lot); err != nil {
			return game.RoomSnapshot{}, err
		}
	}

	return room.SnapshotFor(playerID), nil
}

func (s *RoomService) PlaceBid(_ context.Context, roomID string, playerID string, amount int) (PlaceBidResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	room, err := s.room(roomID)
	if err != nil {
		return PlaceBidResult{}, err
	}
	if err := room.PlaceBid(playerID, amount); err != nil {
		return PlaceBidResult{
			Accepted: false,
			Snapshot: room.SnapshotFor(playerID),
		}, err
	}

	return PlaceBidResult{
		Accepted: true,
		Snapshot: room.SnapshotFor(playerID),
	}, nil
}

func (s *RoomService) SettleRound(_ context.Context, roomID string, playerID string) (SettleRoundResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	room, err := s.room(roomID)
	if err != nil {
		return SettleRoundResult{}, err
	}

	result, err := room.SettleRound()
	if err != nil {
		return SettleRoundResult{}, err
	}

	return SettleRoundResult{
		Result:   result,
		Snapshot: room.SnapshotFor(playerID),
	}, nil
}

func (s *RoomService) StartNextRound(ctx context.Context, roomID string, playerID string) (game.RoomSnapshot, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	room, err := s.room(roomID)
	if err != nil {
		return game.RoomSnapshot{}, err
	}

	lot, err := s.lotProvider.NextLot(ctx, room.SnapshotFor(playerID))
	if err != nil {
		return game.RoomSnapshot{}, err
	}
	if err := room.StartNextRound(lot); err != nil {
		return game.RoomSnapshot{}, err
	}

	return room.SnapshotFor(playerID), nil
}

func (s *RoomService) Snapshot(_ context.Context, roomID string, playerID string) (game.RoomSnapshot, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	room, err := s.room(roomID)
	if err != nil {
		return game.RoomSnapshot{}, err
	}

	return room.SnapshotFor(playerID), nil
}

func (s *RoomService) RoomIDForPlayer(_ context.Context, playerID string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	roomID, ok := s.playerRooms[playerID]
	if !ok {
		return "", ErrPlayerHasNoRoom
	}

	return roomID, nil
}

func (s *RoomService) room(roomID string) (*game.Room, error) {
	room, ok := s.rooms[roomID]
	if !ok {
		return nil, ErrRoomNotFound
	}

	return room, nil
}
