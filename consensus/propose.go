package consensus

import "time"

type ProposeStep struct {
	ticker  TimeoutTicker
	timeout time.Duration
	rs      *RoundState
}

func NewProposeStep(ticker TimeoutTicker, timeout time.Duration, rs *RoundState) *ProposeStep {
	return &ProposeStep{ticker: ticker, timeout: timeout, rs: rs}
}

func (s *ProposeStep) enter(height int64, round int32) {
	s.ticker.SetTimeout(RoundEvent{
		TTL:    s.timeout,
		Height: height,
		Round:  round,
		Step:   RoundStepNewHeight,
	})
}
func (s *ProposeStep) done(height int64, round int32)   {}
func (s *ProposeStep) cancel(height int64, round int32) {}
