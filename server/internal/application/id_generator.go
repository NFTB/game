package application

import (
	"fmt"
	"sync/atomic"
)

type SequentialIDGenerator struct {
	next uint64
}

func NewSequentialIDGenerator() *SequentialIDGenerator {
	return &SequentialIDGenerator{}
}

func (g *SequentialIDGenerator) NewID(prefix string) string {
	value := atomic.AddUint64(&g.next, 1)
	return fmt.Sprintf("%s_%06d", prefix, value)
}
