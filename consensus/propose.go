package consensus

import "time"

type ProposeStep struct {
	ticker   TimeoutTicker
	timeout  time.Duration
	rs       *RoundState
	msgQueue chan Message
}

func NewProposeStep(ticker TimeoutTicker, timeout time.Duration, rs *RoundState, msgQueue chan Message) *ProposeStep {
	return &ProposeStep{ticker: ticker, timeout: timeout, rs: rs, msgQueue: msgQueue}
}

func (s *ProposeStep) enter(height int64, round int32) {
	s.ticker.SetTimeout(RoundEvent{
		TTL:    s.timeout,
		Height: height,
		Round:  round,
		Step:   RoundStepNewHeight,
	})
}
func (s *ProposeStep) complete(height int64, round int32) {}
func (s *ProposeStep) cancel(height int64, round int32)   {}
