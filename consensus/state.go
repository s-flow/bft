package consensus

import (
	"sync"

	"github.com/cometbft/cometbft/state"
	"github.com/cometbft/cometbft/types"
)

type State struct {
	state.State
	mtx sync.RWMutex
}

func NewState(state state.State) *State {
	return &State{State: state}
}

func (cs *State) GetState() state.State {
	cs.mtx.RLock()
	defer cs.mtx.RUnlock()
	return cs.State.Copy()
}

// GetValidators returns a copy of the current validators.
func (cs *State) GetValidators() (int64, []*types.Validator) {
	cs.mtx.RLock()
	defer cs.mtx.RUnlock()
	return cs.State.LastBlockHeight, cs.State.Validators.Copy().Validators
}
