package consensus

import (
	"time"

	"github.com/cometbft/cometbft/libs/log"
	"github.com/cometbft/cometbft/libs/service"
)

var (
	tickTockBufferSize = 10
)

// TimeoutTicker is a timer that schedules timeouts
// conditional on the height/round/step in the timeoutInfo.
// The timeoutInfo.Duration may be non-positive.
type TimeoutTicker interface {
	Start() error
	Stop() error
	Chan() <-chan RoundEvent  // on which to receive a timeout
	SetTimeout(ev RoundEvent) // reset the timer

	SetLogger(log.Logger)
}

// timeoutTicker wraps time.Timer,
// scheduling timeouts only for greater height/round/step
// than what it's already seen.
// Timeouts are scheduled along the tickChan,
// and fired on the tockChan.
type timeoutTicker struct {
	service.BaseService

	timer    *time.Timer
	tickChan chan RoundEvent // for scheduling timeouts
	tockChan chan RoundEvent // for notifying about them
}

// NewTimeoutTicker returns a new TimeoutTicker.
func NewTimeoutTicker() TimeoutTicker {
	tt := &timeoutTicker{
		timer:    time.NewTimer(0),
		tickChan: make(chan RoundEvent, tickTockBufferSize),
		tockChan: make(chan RoundEvent, tickTockBufferSize),
	}
	tt.BaseService = *service.NewBaseService(nil, "TimeoutTicker", tt)
	tt.stopTimer() // don't want to fire until the first scheduled timeout
	return tt
}

// OnStart implements service.Service. It starts the timeout routine.
func (t *timeoutTicker) OnStart() error {
	go t.timeoutRoutine()

	return nil
}

// OnStop implements service.Service. It stops the timeout routine.
func (t *timeoutTicker) OnStop() {
	t.BaseService.OnStop()
	t.stopTimer()
}

// Chan returns a channel on which timeouts are sent.
func (t *timeoutTicker) Chan() <-chan RoundEvent {
	return t.tockChan
}

// ScheduleTimeout schedules a new timeout by sending on the internal tickChan.
// The timeoutRoutine is always available to read from tickChan, so this won't block.
// The scheduling may fail if the timeoutRoutine has already scheduled a timeout for a later height/round/step.
func (t *timeoutTicker) SetTimeout(ev RoundEvent) {
	t.tickChan <- ev
}

//-------------------------------------------------------------

// stop the timer and drain if necessary
func (t *timeoutTicker) stopTimer() {
	// Stop() returns false if it was already fired or was stopped
	if !t.timer.Stop() {
		select {
		case <-t.timer.C:
		default:
			t.Logger.Debug("Timer already stopped")
		}
	}
}

// send on tickChan to start a new timer.
// timers are interupted and replaced by new ticks from later steps
// timeouts of 0 on the tickChan will be immediately relayed to the tockChan
func (t *timeoutTicker) timeoutRoutine() {
	t.Logger.Debug("Starting timeout routine")
	var ev RoundEvent
	for {
		select {
		case newev := <-t.tickChan:
			t.Logger.Debug("Received tick", "old_ev", ev, "new_ev", newev)

			// ignore tickers for old height/round/step
			if newev.Height < ev.Height {
				continue
			} else if newev.Height == ev.Height {
				if newev.Round < ev.Round {
					continue
				} else if newev.Round == ev.Round {
					if ev.Step > 0 && newev.Step <= ev.Step {
						continue
					}
				}
			}

			// stop the last timer
			t.stopTimer()

			// update timeoutInfo and reset timer
			// NOTE time.Timer allows duration to be non-positive
			ev = newev
			t.timer.Reset(ev.TTL)
			t.Logger.Debug("Scheduled timeout", "ttl", ev.TTL, "height", ev.Height, "round", ev.Round, "step", ev.Step)
		case <-t.timer.C:
			t.Logger.Info("Timed out", "dur", ev.TTL, "height", ev.Height, "round", ev.Round, "step", ev.Step)
			// go routine here guarantees timeoutRoutine doesn't block.
			// Determinism comes from playback in the receiveRoutine.
			// We can eliminate it by merging the timeoutRoutine into receiveRoutine
			//  and managing the timeouts ourselves with a millisecond ticker
			go func(toe RoundEvent) { t.tockChan <- toe }(ev)
		case <-t.Quit():
			return
		}
	}
}
