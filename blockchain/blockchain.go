package blockchain

import (
	"sync"

	sm "github.com/cometbft/cometbft/state"
	"github.com/cometbft/cometbft/types"
)

type Blockchain struct {
	blockStore sm.BlockStore     // store blocks and commits
	blockExec  *sm.BlockExecutor // create and execute blocks
	mtx        sync.RWMutex
}

func NewBlockchain(bs sm.BlockStore, be *sm.BlockExecutor) *Blockchain {
	return &Blockchain{blockStore: bs, blockExec: be}
}

func (bc *Blockchain) SetEventBus(b *types.EventBus) {
	bc.blockExec.SetEventBus(b)
}

func (bc *Blockchain) LoadCommit(height int64) *types.Commit {
	bc.mtx.RLock()
	defer bc.mtx.RUnlock()

	if height == bc.blockStore.Height() {
		return bc.blockStore.LoadSeenCommit(height)
	}

	return bc.blockStore.LoadBlockCommit(height)
}

func (bc *Blockchain) LoadBlockMeta(height int64) *types.BlockMeta {
	return bc.blockStore.LoadBlockMeta(height)
}
