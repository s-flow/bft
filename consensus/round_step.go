package consensus

type RoundStepType uint8

const (
	RoundStepNewHeight RoundStepType = iota
	RoundStepPropose
	RoundStepPrevote
	RoundStepPrecommit
	RoundStepCommit
)

func (rs RoundStepType) String() string {
	switch rs {
	case RoundStepNewHeight:
		return "RoundStepPropose"
	case RoundStepPropose:
		return "RoundStepPropose"
	case RoundStepPrevote:
		return "RoundStepPrevote"
	case RoundStepPrecommit:
		return "RoundStepPrecommit"
	case RoundStepCommit:
		return "RoundStepCommit"
	default:
		return "RoundStepUnknown" // Cannot panic.
	}
}

type RoundStepSet []RoundStep

type RoundStep interface {
	enter(height int64, round int32)
	complete(height int64, round int32)
	cancel(height int64, round int32)
}
