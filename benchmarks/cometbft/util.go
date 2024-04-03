package cometbft

import "github.com/zeu5/dist-rl-testing/core"

func copyRequests(in []CometRequest) []CometRequest {
	out := make([]CometRequest, len(in))
	for i, r := range in {
		out[i] = r.Copy()
	}
	return out
}

func HeightBound(height int) func(core.PState) bool {
	return func(p core.PState) bool {
		cs := p.(*CometClusterState)
		for _, ns := range cs.NodeStates {
			if ns.Height > height {
				return true
			}
		}
		return false
	}
}
