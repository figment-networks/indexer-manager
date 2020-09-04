package cosmos

import (
	"sync"

	"github.com/figment-networks/cosmos-indexer/structs"
)

type SimpleBlockCache struct {
	space  map[uint64]structs.Block
	blocks chan *structs.Block
	l      sync.RWMutex
}

func NewSimpleBlockCache(cap int) *SimpleBlockCache {
	return &SimpleBlockCache{
		space:  make(map[uint64]structs.Block),
		blocks: make(chan *structs.Block, cap),
	}
}

func (sbc *SimpleBlockCache) Add(bl structs.Block) {
	sbc.l.Lock()
	defer sbc.l.Unlock()

	_, ok := sbc.space[bl.Height]
	if ok {
		return
	}

	sbc.space[bl.Height] = bl
	select {
	case sbc.blocks <- &bl:
	default:
		oldBlock := <-sbc.blocks
		if oldBlock != nil {
			delete(sbc.space, oldBlock.Height)
		}
		sbc.blocks <- &bl
	}

}

func (sbc *SimpleBlockCache) Get(height uint64) (bl structs.Block, ok bool) {
	sbc.l.RLock()
	defer sbc.l.RUnlock()

	bl, ok = sbc.space[bl.Height]
	return bl, ok
}
