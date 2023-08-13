package consensus

import (
	cmttime "github.com/cometbft/cometbft/types/time"
)

type NewHeightStep struct {
	ticker TimeoutTicker
	rs     *RoundState
}

func NewNewHeightStep(ticker TimeoutTicker, rs *RoundState) *NewHeightStep {
	return &NewHeightStep{ticker: ticker, rs: rs}
}

func (s *NewHeightStep) enter(height int64, round int32) {
	sleep := s.rs.StartTime.Sub(cmttime.Now())
	s.ticker.SetTimeout(RoundEvent{
		TTL:    sleep,
		Height: height,
		Round:  round,
		Step:   RoundStepNewHeight,
	})
}

func (s *NewHeightStep) done(height int64, round int32)   {}
func (s *NewHeightStep) cancel(height int64, round int32) {}
