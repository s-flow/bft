package consensus

import (
	"github.com/cometbft/cometbft/types"
	"github.com/s-flow/simple-bft/signer"
)

type Validator interface {
	SignVote(chainID string, vote *types.Vote)
	SignProposal(chainID string, proposal *types.Proposal)
	CreateProposal() *types.Proposal
}

type LocalValidator struct {
	signer signer.Signer
}

func NewLocalValidator(signer signer.Signer) *LocalValidator {
	return &LocalValidator{signer: signer}
}

func (v *LocalValidator) SignVote(chainID string, vote *types.Vote) {

}

func (v *LocalValidator) SignProposal(chainID string, proposal *types.Proposal) {

}

func (v *LocalValidator) CreateProposal() *types.Proposal {

	return nil
}
