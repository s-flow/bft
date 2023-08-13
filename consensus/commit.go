package consensus

import "time"

type CommitStep struct {
	ticker  TimeoutTicker
	timeout time.Duration
	rs      *RoundState
}

func NewCommitStep(ticker TimeoutTicker, timeout time.Duration, rs *RoundState) *CommitStep {
	return &CommitStep{ticker: ticker, timeout: timeout, rs: rs}
}

func (s *CommitStep) enter(height int64, round int32) {
	s.ticker.SetTimeout(RoundEvent{
		TTL:    s.timeout,
		Height: height,
		Round:  round,
		Step:   RoundStepCommit,
	})
}
func (s *CommitStep) done(height int64, round int32)   {}
func (s *CommitStep) cancel(height int64, round int32) {}
