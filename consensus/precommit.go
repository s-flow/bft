package consensus

import "time"

type PrecommitStep struct {
	ticker   TimeoutTicker
	timeout  time.Duration
	rs       *RoundState
	msgQueue chan Message
}

func NewPrecommitStep(ticker TimeoutTicker, timeout time.Duration, rs *RoundState, msgQueue chan Message) *PrecommitStep {
	return &PrecommitStep{ticker: ticker, timeout: timeout, rs: rs, msgQueue: msgQueue}
}

func (s *PrecommitStep) enter(height int64, round int32) {
	s.ticker.SetTimeout(RoundEvent{
		TTL:    s.timeout,
		Height: height,
		Round:  round,
		Step:   RoundStepNewHeight,
	})
}
func (ps *PrecommitStep) complete(height int64, round int32) {}
func (ps *PrecommitStep) cancel(height int64, round int32)   {}
