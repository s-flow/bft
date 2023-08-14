package consensus

import "time"

type PrevoteStep struct {
	ticker   TimeoutTicker
	timeout  time.Duration
	rs       *RoundState
	msgQueue chan Message
}

func NewPrevoteStep(ticker TimeoutTicker, timeout time.Duration, rs *RoundState, msgQueue chan Message) *PrevoteStep {
	return &PrevoteStep{ticker: ticker, timeout: timeout, rs: rs, msgQueue: msgQueue}
}

func (s *PrevoteStep) enter(height int64, round int32) {
	s.ticker.SetTimeout(RoundEvent{
		TTL:    s.timeout,
		Height: height,
		Round:  round,
		Step:   RoundStepNewHeight,
	})
}
func (s *PrevoteStep) complete(height int64, round int32) {}
func (s *PrevoteStep) cancel(height int64, round int32)   {}
