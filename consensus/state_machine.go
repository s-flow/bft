package consensus

import (
	"bytes"
	"errors"
	"fmt"
	"sync"
	"time"

	cfg "github.com/cometbft/cometbft/config"
	cstypes "github.com/cometbft/cometbft/consensus/types"
	"github.com/cometbft/cometbft/libs/log"
	cmtmath "github.com/cometbft/cometbft/libs/math"
	"github.com/cometbft/cometbft/libs/service"
	"github.com/cometbft/cometbft/p2p"
	"github.com/cometbft/cometbft/types"
	cmttime "github.com/cometbft/cometbft/types/time"
	"github.com/s-flow/simple-bft/blockchain"
)

// Consensus sentinel errors
var (
	ErrInvalidProposalSignature   = errors.New("error invalid proposal signature")
	ErrInvalidProposalPOLRound    = errors.New("error invalid proposal POL round")
	ErrAddingVote                 = errors.New("error adding vote")
	ErrSignatureFoundInPastBlocks = errors.New("found signature from the same key")

	errPubKeyIsNotSet = errors.New("pubkey is not set. Look for \"Can't get private validator pubkey\" errors")
)

const MaxMsgQueueSize = 1000

type StateMachine struct {
	service.BaseService
	config *cfg.ConsensusConfig

	vm        VoteManager
	validator Validator

	blockchain *blockchain.Blockchain

	state State // chain state

	roundStepSet RoundStepSet
	roundState   RoundState

	msgQueue chan Message

	eventBus *types.EventBus

	timeoutTicker TimeoutTicker

	mtx sync.RWMutex
}

func NewStateMachine(
	config *cfg.ConsensusConfig,
	state State,
	bc *blockchain.Blockchain,
	validator Validator,
	eb *types.EventBus,
) *StateMachine {
	sm := &StateMachine{
		config:        config,
		validator:     validator,
		blockchain:    bc,
		eventBus:      eb,
		timeoutTicker: NewTimeoutTicker(),
		msgQueue:      make(chan Message, MaxMsgQueueSize),
	}

	sm.updateState(state)
	sm.setupRoundStepSet()

	sm.BaseService = *service.NewBaseService(nil, "StateMachine", sm)
	return sm
}

func (sm *StateMachine) OnStart() error {

	// turn on timeout ticker
	if err := sm.timeoutTicker.Start(); err != nil {
		return err
	}

	go sm.routine()
	return nil
}

func (sm *StateMachine) setupRoundStepSet() {
	sm.roundStepSet = RoundStepSet{
		NewNewHeightStep(sm.timeoutTicker, &sm.roundState),
		NewProposeStep(sm.timeoutTicker, sm.config.TimeoutPropose, &sm.roundState),
		NewPrevoteStep(sm.timeoutTicker, sm.config.TimeoutPrevote, &sm.roundState),
		NewPrecommitStep(sm.timeoutTicker, sm.config.TimeoutPrecommit, &sm.roundState),
		NewCommitStep(sm.timeoutTicker, sm.config.TimeoutCommit, &sm.roundState),
	}
}

func (sm *StateMachine) enter(height int64, round int32, step RoundStepType) {
	sm.roundStepSet[step].enter(height, round)
}

func (sm *StateMachine) done(height int64, round int32, step RoundStepType) {
	sm.roundStepSet[step].done(height, round)
}

func (sm *StateMachine) setTimeout(ttl time.Duration, height int64, round int32, step RoundStepType) {
	sm.timeoutTicker.SetTimeout(RoundEvent{ttl, height, round, step})
}

// TODO: done channel
func (sm *StateMachine) routine() {
	defer func() {
		if r := recover(); r != nil {
			// stop gracefully
		}
	}()

	for {
		select {
		case msg := <-sm.msgQueue:
			sm.handleMsg(msg, sm.roundState)
		case timeout := <-sm.timeoutTicker.Chan():
			sm.handleTimeout(timeout, sm.roundState)
		case <-sm.Quit():
			return
		}
	}
}

func (sm *StateMachine) updateState(s State) {
	sm.mtx.Lock()
	defer sm.mtx.Unlock()
	sm.state = s
}

func (sm *StateMachine) SetValidator(validator Validator) {
	sm.mtx.Lock()
	defer sm.mtx.Unlock()

	sm.validator = validator
}

func (sm *StateMachine) addProposalBlockPart(event *BlockPartEvent, peerID p2p.ID) (bool, error) {
	sm.mtx.Lock()
	defer sm.mtx.Unlock()

	return sm.roundState.ProposalBlockParts.AddPart(event.Part)
}

func (sm *StateMachine) setProposal(proposal *types.Proposal) error {
	sm.mtx.Lock()
	defer sm.mtx.Unlock()

	// verify signature
	p := proposal.ToProto()
	if !sm.roundState.Validators.GetProposer().PubKey.VerifySignature(
		types.ProposalSignBytes(sm.state.ChainID, p), proposal.Signature,
	) {
		return ErrInvalidProposalSignature
	}

	sm.roundState.Proposal = proposal

	return nil
}

func (sm *StateMachine) createProposal() {

}

func (sm *StateMachine) commit() {

}

func (sm *StateMachine) handleMsg(msg Message) {
	ev := msg.Event
	switch event := ev.(type) {
	case *ProposalEvent:
		if err := sm.setProposal(event.Proposal); err != nil {
			// logging
		}
	case *BlockPartEvent:
		added, err = sm.addProposalBlockPart(event, msg.PeerID)
		if added {
			sm.handleProposal()
		}
	case *VoteEvent:

	default:
		return
	}
}

func (sm *StateMachine) handleTimeout(re RoundEvent, rs RoundState) {
	if !re.IsValid(rs) {
		return
	}

	switch re.Step {
	case RoundStepNewHeight:
		sm.done(re.Height, 0, RoundStepNewHeight)

	case RoundStepPropose:
		if err := sm.eventBus.PublishEventTimeoutPropose(sm.roundState.RoundStateEvent()); err != nil {
			sm.Logger.Error("failed publishing timeout propose", "err", err)
		}
		sm.enter(re.Height, re.Round, RoundStepPrevote)

	case RoundStepPrevote:
		if err := sm.eventBus.PublishEventTimeoutWait(sm.roundState.RoundStateEvent()); err != nil {
			sm.Logger.Error("failed publishing timeout wait", "err", err)
		}
		sm.enter(re.Height, re.Round, RoundStepPrecommit)

	case RoundStepPrecommit:
		if err := sm.eventBus.PublishEventTimeoutWait(sm.roundState.RoundStateEvent()); err != nil {
			sm.Logger.Error("failed publishing timeout wait", "err", err)
		}

		sm.enter(re.Height, re.Round, RoundStepPrecommit)
		sm.newRound(re.Height, re.Round+1)

	default:
		panic(fmt.Sprintf("invalid timeout step: %v", re.Step))
	}
}

func (cs *StateMachine) handleProposal(height int64) {

}

func (cs *StateMachine) newRound(height int64, round int32) {
	logger := cs.Logger.With("height", height, "round", round)
	if cs.roundState.Height != height || round < cs.roundState.Round || (cs.roundState.Round == round && cs.roundState.Step != cstypes.RoundStepNewHeight) {
		logger.Debug(
			"entering new round with invalid args",
			"current", log.NewLazySprintf("%v/%v/%v", cs.roundState.Height, cs.roundState.Round, cs.roundState.Step),
		)
		return
	}

	if now := cmttime.Now(); cs.roundState.StartTime.After(now) {
		logger.Debug("need to set a buffer and log message here for sanity", "start_time", cs.StartTime, "now", now)
	}

	logger.Debug("entering new round", "current", log.NewLazySprintf("%v/%v/%v", cs.Height, cs.Round, cs.Step))

	// increment validators if necessary
	validators := cs.roundState.Validators
	if cs.roundState.Round < round {
		validators = validators.Copy()
		validators.IncrementProposerPriority(cmtmath.SafeSubInt32(round, cs.Round))
	}

	// Setup new round
	// we don't fire newStep for this step,
	// but we fire an event, so update the round step first
	cs.updateRoundStep(round, RoundStepNewHeight)
	cs.roundState.Validators = validators
	// If round == 0, we've already reset these upon new height, and meanwhile
	// we might have received a proposal for round 0.
	if round != 0 {
		logger.Debug("resetting proposal info")
		cs.roundState.Proposal = nil
		cs.roundState.ProposalBlock = nil
		cs.roundState.ProposalBlockParts = nil
	}

	cs.roundState.Votes.SetRound(cmtmath.SafeAddInt32(round, 1)) // also track next round (round+1) to allow round-skipping
	cs.roundState.TriggeredTimeoutPrecommit = false

	if err := cs.eventBus.PublishEventNewRound(cs.roundState.NewRoundEvent()); err != nil {
		cs.Logger.Error("failed publishing new round", "err", err)
	}
	// Wait for txs to be available in the mempool
	// before we enterPropose in round 0. If the last block changed the app hash,
	// we may need an empty "proof" block, and enterPropose immediately.
	waitForTxs := cs.config.WaitForTxs() && round == 0 && !cs.needProofBlock(height)
	if waitForTxs {
		if cs.config.CreateEmptyBlocksInterval > 0 {
			cs.setTimeout(cs.config.CreateEmptyBlocksInterval, height, round,
				RoundStepNewHeight)
		}
	} else {
		cs.enter(height, round, RoundStepPropose)
	}
}

// needProofBlock returns true on the first height (so the genesis app hash is signed right away)
// and where the last block (height-1) caused the app hash to change
func (sm *StateMachine) needProofBlock(height int64) bool {
	if height == sm.state.InitialHeight {
		return true
	}

	lastBlockMeta := sm.blockchain.LoadBlockMeta(height - 1)
	if lastBlockMeta == nil {
		// See https://github.com/cometbft/cometbft/issues/370
		sm.Logger.Info("short-circuited needProofBlock", "height", height, "InitialHeight", cs.state.InitialHeight)
		return true
	}

	return !bytes.Equal(sm.state.AppHash, lastBlockMeta.Header.AppHash)
}

func (sm *StateMachine) updateHeight(height int64) {
	sm.roundState.Height = height
}

func (sm *StateMachine) updateRoundStep(round int32, step RoundStepType) {
	sm.roundState.Round = round
	sm.roundState.Step = step
}
