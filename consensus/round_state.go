package consensus

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	cstypes "github.com/cometbft/cometbft/consensus/types"
	"github.com/cometbft/cometbft/libs/bytes"
	cmtjson "github.com/cometbft/cometbft/libs/json"
	"github.com/cometbft/cometbft/types"
)

// round state warpping
type RoundState struct {
	Height    int64         `json:"height"` // Height we are working on
	Round     int32         `json:"round"`
	Step      RoundStepType `json:"step"`
	StartTime time.Time     `json:"start_time"`

	// Subjective time when +2/3 precommits for Block at Round were found
	CommitTime         time.Time           `json:"commit_time"`
	Validators         *types.ValidatorSet `json:"validators"`
	Proposal           *types.Proposal     `json:"proposal"`
	ProposalBlock      *types.Block        `json:"proposal_block"`
	ProposalBlockParts *types.PartSet      `json:"proposal_block_parts"`
	LockedRound        int32               `json:"locked_round"`
	LockedBlock        *types.Block        `json:"locked_block"`
	LockedBlockParts   *types.PartSet      `json:"locked_block_parts"`

	// The variables below starting with "Valid..." derive their name from
	// the algorithm presented in this paper:
	// [The latest gossip on BFT consensus](https://arxiv.org/abs/1807.04938).
	// Therefore, "Valid...":
	//   * means that the block or round that the variable refers to has
	//     received 2/3+ non-`nil` prevotes (a.k.a. a *polka*)
	//   * has nothing to do with whether the Application returned "Accept" in its
	//     response to `ProcessProposal`, or "Reject"

	// Last known round with POL for non-nil valid block.
	ValidRound int32        `json:"valid_round"`
	ValidBlock *types.Block `json:"valid_block"` // Last known block of POL mentioned above.

	// Last known block parts of POL mentioned above.
	ValidBlockParts           *types.PartSet         `json:"valid_block_parts"`
	Votes                     *cstypes.HeightVoteSet `json:"votes"`
	CommitRound               int32                  `json:"commit_round"` //
	LastCommit                *types.VoteSet         `json:"last_commit"`  // Last precommits at Height-1
	LastValidators            *types.ValidatorSet    `json:"last_validators"`
	TriggeredTimeoutPrecommit bool                   `json:"triggered_timeout_precommit"`

	mtx sync.RWMutex `json:"-"`
}

// GetLastHeight returns the last height committed.
// If there were no blocks, returns 0.
func (rs *RoundState) GetLastHeight() int64 {
	rs.mtx.RLock()
	defer rs.mtx.RUnlock()
	return rs.Height - 1
}

// GetState returns a shallow copy of the internal consensus state.
func (rs *RoundState) GetState() *RoundState {
	rs.mtx.RLock()
	state := rs // copy
	rs.mtx.RUnlock()
	return state
}

// JSON returns a json of RoundState.
func (rs *RoundState) JSON() ([]byte, error) {
	rs.mtx.RLock()
	defer rs.mtx.RUnlock()
	return cmtjson.Marshal(rs)
}

// SimpleJSON returns a json of RoundStateSimple
func (rs *RoundState) SimpleJSON() ([]byte, error) {
	rs.mtx.RLock()
	defer rs.mtx.RUnlock()
	return cmtjson.Marshal(rs.RoundStateSimple())
}

// Compress the RoundState to RoundStateSimple
func (rs *RoundState) RoundStateSimple() RoundStateSimple {
	votesJSON, err := rs.Votes.MarshalJSON()
	if err != nil {
		panic(err)
	}

	addr := rs.Validators.GetProposer().Address
	idx, _ := rs.Validators.GetByAddress(addr)

	return RoundStateSimple{
		HeightRoundStep:   fmt.Sprintf("%d/%d/%d", rs.Height, rs.Round, rs.Step),
		StartTime:         rs.StartTime,
		ProposalBlockHash: rs.ProposalBlock.Hash(),
		LockedBlockHash:   rs.LockedBlock.Hash(),
		ValidBlockHash:    rs.ValidBlock.Hash(),
		Votes:             votesJSON,
		Proposer: types.ValidatorInfo{
			Address: addr,
			Index:   idx,
		},
	}
}

// Compressed version of the RoundState for use in RPC
type RoundStateSimple struct {
	HeightRoundStep   string              `json:"height/round/step"`
	StartTime         time.Time           `json:"start_time"`
	ProposalBlockHash bytes.HexBytes      `json:"proposal_block_hash"`
	LockedBlockHash   bytes.HexBytes      `json:"locked_block_hash"`
	ValidBlockHash    bytes.HexBytes      `json:"valid_block_hash"`
	Votes             json.RawMessage     `json:"height_vote_set"`
	Proposer          types.ValidatorInfo `json:"proposer"`
}

// NewRoundEvent returns the RoundState with proposer information as an event.
func (rs *RoundState) NewRoundEvent() types.EventDataNewRound {
	addr := rs.Validators.GetProposer().Address
	idx, _ := rs.Validators.GetByAddress(addr)

	return types.EventDataNewRound{
		Height: rs.Height,
		Round:  rs.Round,
		Step:   rs.Step.String(),
		Proposer: types.ValidatorInfo{
			Address: addr,
			Index:   idx,
		},
	}
}

// CompleteProposalEvent returns information about a proposed block as an event.
func (rs *RoundState) CompleteProposalEvent() types.EventDataCompleteProposal {
	// We must construct BlockID from ProposalBlock and ProposalBlockParts
	// cs.Proposal is not guaranteed to be set when this function is called
	blockID := types.BlockID{
		Hash:          rs.ProposalBlock.Hash(),
		PartSetHeader: rs.ProposalBlockParts.Header(),
	}

	return types.EventDataCompleteProposal{
		Height:  rs.Height,
		Round:   rs.Round,
		Step:    rs.Step.String(),
		BlockID: blockID,
	}
}

// RoundStateEvent returns the H/R/S of the RoundState as an event.
func (rs *RoundState) RoundStateEvent() types.EventDataRoundState {
	return types.EventDataRoundState{
		Height: rs.Height,
		Round:  rs.Round,
		Step:   rs.Step.String(),
	}
}

// String returns a string
func (rs *RoundState) String() string {
	return rs.StringIndented("")
}

// StringIndented returns a string
func (rs *RoundState) StringIndented(indent string) string {
	return fmt.Sprintf(`RoundState{
%s  H:%v R:%v S:%v
%s  StartTime:     %v
%s  CommitTime:    %v
%s  Validators:    %v
%s  Proposal:      %v
%s  ProposalBlock: %v %v
%s  LockedRound:   %v
%s  LockedBlock:   %v %v
%s  ValidRound:    %v
%s  ValidBlock:    %v %v
%s  Votes:         %v
%s  LastCommit:    %v
%s  LastValidators:%v
%s}`,
		indent, rs.Height, rs.Round, rs.Step,
		indent, rs.StartTime,
		indent, rs.CommitTime,
		indent, rs.Validators.StringIndented(indent+"  "),
		indent, rs.Proposal,
		indent, rs.ProposalBlockParts.StringShort(), rs.ProposalBlock.StringShort(),
		indent, rs.LockedRound,
		indent, rs.LockedBlockParts.StringShort(), rs.LockedBlock.StringShort(),
		indent, rs.ValidRound,
		indent, rs.ValidBlockParts.StringShort(), rs.ValidBlock.StringShort(),
		indent, rs.Votes.StringIndented(indent+"  "),
		indent, rs.LastCommit.StringShort(),
		indent, rs.LastValidators.StringIndented(indent+"  "),
		indent)
}

// StringShort returns a string
func (rs *RoundState) StringShort() string {
	return fmt.Sprintf(`RoundState{H:%v R:%v S:%v ST:%v}`,
		rs.Height, rs.Round, rs.Step, rs.StartTime)
}
