package consensus

import (
	"errors"
	"fmt"
	"time"

	"github.com/cometbft/cometbft/libs/bits"
	"github.com/cometbft/cometbft/p2p"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	"github.com/cometbft/cometbft/types"
)

type Event interface {
	ValidateBasic() error
}

type Message struct {
	Event  Event  `json:"event"`
	PeerID p2p.ID `json:"peer_key"`
}

type RoundEvent struct {
	TTL    time.Duration `json:"ttl"`
	Height int64         `json:"height"`
	Round  int32         `json:"round"`
	Step   RoundStepType `json:"step"`
}

func (re *RoundEvent) IsValid(rs RoundState) bool {
	if re.Height != rs.Height || re.Round < rs.Round ||
		(re.Round == rs.Round && re.Step < rs.Step) {
		return false
	}

	return true
}

type ProposalEvent struct {
	Proposal *types.Proposal
}

func (m *ProposalEvent) ValidateBasic() error {
	return m.Proposal.ValidateBasic()
}

// String returns a string representation.
func (m *ProposalEvent) String() string {
	return fmt.Sprintf("[Proposal %v]", m.Proposal)
}

type ProposalPOLEvent struct {
	Height           int64
	ProposalPOLRound int32
	ProposalPOL      *bits.BitArray
}

// ValidateBasic performs basic validation.
func (m *ProposalPOLEvent) ValidateBasic() error {
	if m.Height < 0 {
		return errors.New("negative Height")
	}
	if m.ProposalPOLRound < 0 {
		return errors.New("negative ProposalPOLRound")
	}
	if m.ProposalPOL.Size() == 0 {
		return errors.New("empty ProposalPOL bit array")
	}
	if m.ProposalPOL.Size() > types.MaxVotesCount {
		return fmt.Errorf("proposalPOL bit array is too big: %d, max: %d", m.ProposalPOL.Size(), types.MaxVotesCount)
	}
	return nil
}

// String returns a string representation.
func (m *ProposalPOLEvent) String() string {
	return fmt.Sprintf("[ProposalPOL H:%v POLR:%v POL:%v]", m.Height, m.ProposalPOLRound, m.ProposalPOL)
}

// BlockPartMessage is sent when gossipping a piece of the proposed block.
type BlockPartEvent struct {
	Height int64
	Round  int32
	Part   *types.Part
}

// ValidateBasic performs basic validation.
func (m *BlockPartEvent) ValidateBasic() error {
	if m.Height < 0 {
		return errors.New("negative Height")
	}
	if m.Round < 0 {
		return errors.New("negative Round")
	}
	if err := m.Part.ValidateBasic(); err != nil {
		return fmt.Errorf("wrong Part: %v", err)
	}
	return nil
}

// String returns a string representation.
func (m *BlockPartEvent) String() string {
	return fmt.Sprintf("[BlockPart H:%v R:%v P:%v]", m.Height, m.Round, m.Part)
}

// VoteMessage is sent when voting for a proposal (or lack thereof).
type RoundTriggerEvent struct {
	Event RoundEvent
}

// ValidateBasic checks whether the vote within the message is well-formed.
func (m *RoundTriggerEvent) ValidateBasic() error {
	// return m.Event.ValidateBasic()
}

// String returns a string representation.
func (m *RoundTriggerEvent) String() string {
	return fmt.Sprintf("[RoundTriggerEvent %v]", m.Event)
}

// VoteMessage is sent when voting for a proposal (or lack thereof).
type VoteEvent struct {
	Vote *types.Vote
}

// ValidateBasic checks whether the vote within the message is well-formed.
func (m *VoteEvent) ValidateBasic() error {
	return m.Vote.ValidateBasic()
}

// String returns a string representation.
func (m *VoteEvent) String() string {
	return fmt.Sprintf("[Vote %v]", m.Vote)
}

//-------------------------------------

// HasVoteMessage is sent to indicate that a particular vote has been received.
type HasVoteEvent struct {
	Height int64
	Round  int32
	Type   cmtproto.SignedMsgType
	Index  int32
}

// ValidateBasic performs basic validation.
func (m *HasVoteEvent) ValidateBasic() error {
	if m.Height < 0 {
		return errors.New("negative Height")
	}
	if m.Round < 0 {
		return errors.New("negative Round")
	}
	if !types.IsVoteTypeValid(m.Type) {
		return errors.New("invalid Type")
	}
	if m.Index < 0 {
		return errors.New("negative Index")
	}
	return nil
}

// String returns a string representation.
func (m *HasVoteEvent) String() string {
	return fmt.Sprintf("[HasVote VI:%v V:{%v/%02d/%v}]", m.Index, m.Height, m.Round, m.Type)
}
