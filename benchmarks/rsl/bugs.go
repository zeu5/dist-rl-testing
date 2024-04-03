package rsl

import (
	"github.com/zeu5/dist-rl-testing/analysis"
	"github.com/zeu5/dist-rl-testing/core"
)

func inconsistentLogs() func(*core.Trace) bool {
	return func(trace *core.Trace) bool {
		for i := 0; i < trace.Len(); i++ {
			s := trace.Step(i)
			ps, ok := s.State.(*core.PartitionState)
			if !ok {
				continue
			}
			if inconsistentLogState(ps) {
				return true
			}
		}

		return false
	}
}

func inconsistentLogState(state *core.PartitionState) bool {
	maxLogLength := 0
	decidedLogs := make(map[int][]Command)
	for node, rs := range state.NodeStates {
		ls := rs.(LocalState)
		decidedLogs[node] = make([]Command, len(ls.Log.Decided))
		for i, c := range ls.Log.Decided {
			decidedLogs[node][i] = c.Copy()
		}
		if ls.Log.NumDecided() > maxLogLength {
			maxLogLength = ls.Log.NumDecided()
		}
	}
	for j := 0; j < maxLogLength; j++ {
		commands := make(map[string]Command)
		for _, decided := range decidedLogs {
			if j < len(decided) {
				commands[decided[j].Hash()] = decided[j].Copy()
			}
		}
		if len(commands) > 1 {
			return true
		}
	}
	return false
}

func multiplePrimaries() func(*core.Trace) bool {
	return func(t *core.Trace) bool {
		for i := 0; i < t.Len(); i++ {
			s := t.Step(i)
			ps, ok := s.State.(*core.PartitionState)
			if !ok {
				continue
			}
			primaries := make(map[int]int)
			for node, rs := range ps.NodeStates {
				ls := rs.(LocalState)
				if ls.State == StateStablePrimary {
					ballot := ls.MaxAcceptedProposal.Ballot.Num
					if _, ok := primaries[ballot]; ok {
						// There is already a primary for this ballot number
						return true
					}
					primaries[ballot] = node
				}
			}
		}

		return false
	}
}

var InconsistentLogs = analysis.BugSpec{
	Name:  "InconsistentLogs",
	Check: inconsistentLogs(),
}

var MultiplePrimaries = analysis.BugSpec{
	Name:  "MultiplePrimaries",
	Check: multiplePrimaries(),
}
