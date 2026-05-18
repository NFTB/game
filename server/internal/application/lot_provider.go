package application

import (
	"context"
	"sync"

	"bidking/server/internal/game"
)

type StaticLotProvider struct {
	mu   sync.Mutex
	next int
	lots []game.Lot
}

func NewStaticLotProvider(lots []game.Lot) *StaticLotProvider {
	cloned := make([]game.Lot, 0, len(lots))
	for _, lot := range lots {
		cloned = append(cloned, cloneLot(lot))
	}

	return &StaticLotProvider{lots: cloned}
}

func (p *StaticLotProvider) NextLot(_ context.Context, _ game.RoomSnapshot) (game.Lot, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if len(p.lots) == 0 {
		return game.Lot{}, game.ErrInvalidLot
	}

	lot := cloneLot(p.lots[p.next%len(p.lots)])
	p.next++
	return lot, nil
}

func cloneLot(lot game.Lot) game.Lot {
	lot.Items = append([]game.Item(nil), lot.Items...)
	return lot
}
