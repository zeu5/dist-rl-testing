package rsl

import "github.com/zeu5/dist-rl-testing/core"

func ColorState() core.KVPainter {
	return wrapColor(func(ls LocalState) (string, interface{}) {
		return "state", string(ls.State)
	})
}

func ColorBoundedBallot(bound int) core.KVPainter {
	return wrapColor(func(ls LocalState) (string, interface{}) {
		b := ls.MaxAcceptedProposal.Ballot.Num
		if b > bound {
			b = bound
		}
		return "boundedBallot", b
	})
}

func ColorBallot() core.KVPainter {
	return wrapColor(func(ls LocalState) (string, interface{}) {
		return "ballot", ls.MaxAcceptedProposal.Ballot.Num
	})
}

func ColorDecree() core.KVPainter {
	return wrapColor(func(ls LocalState) (string, interface{}) {
		return "decree", ls.MaxAcceptedProposal.Decree
	})
}

func ColorDecided() core.KVPainter {
	return wrapColor(func(ls LocalState) (string, interface{}) {
		return "decided", ls.Decided
	})
}

func ColorLogLength() core.KVPainter {
	return wrapColor(func(ls LocalState) (string, interface{}) {
		return "logLength", ls.Log.NumDecided()
	})
}

func wrapColor(f func(LocalState) (string, interface{})) core.KVPainter {
	return func(s core.NState) (string, interface{}) {
		ns := s.(LocalState)
		return f(ns)
	}

}
