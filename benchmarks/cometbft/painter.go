package cometbft

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"

	"github.com/zeu5/dist-rl-testing/core"
)

func wrapPainter(f func(*CometNodeState) (string, interface{})) core.KVPainter {
	return func(n core.NState) (string, interface{}) {
		return f(n.(*CometNodeState))
	}
}

func ColorHRS() core.KVPainter {
	return wrapPainter(func(s *CometNodeState) (string, interface{}) {
		return "hrs", s.HeightRoundStep
	})
}

func ColorHeight() core.KVPainter {
	return wrapPainter(func(cns *CometNodeState) (string, interface{}) {
		return "height", cns.Height
	})
}

func ColorStep() core.KVPainter {
	return wrapPainter(func(cns *CometNodeState) (string, interface{}) {
		return "step", cns.Step
	})
}

func ColorRound() core.KVPainter {
	return wrapPainter(func(cns *CometNodeState) (string, interface{}) {
		return "round", cns.Round
	})
}

func ColorProposal() core.KVPainter {
	return wrapPainter(func(s *CometNodeState) (string, interface{}) {
		return "have_proposal", s.ProposalBlockHash != ""
	})
}

func ColorLockedValue() core.KVPainter {
	return wrapPainter(func(s *CometNodeState) (string, interface{}) {
		return "locked", s.LockedBlockHash
	})
}

func ColorValidValue() core.KVPainter {
	return wrapPainter(func(s *CometNodeState) (string, interface{}) {
		return "valid", s.ValidBlockHash
	})
}

func ColorNumVotes() core.KVPainter {
	return wrapPainter(func(s *CometNodeState) (string, interface{}) {
		votes := make([][]int, 0)
		for _, v := range s.Votes {
			prevotes := 0
			precommits := 0
			for _, pv := range v.Prevotes {
				if pv != "nil-Vote" {
					prevotes += 1
				}
			}
			for _, pcv := range v.Precommits {
				if pcv != "nil-Vote" {
					precommits += 1
				}
			}
			votes = append(votes, []int{prevotes, precommits})
		}
		return "num_votes", votes
	})
}

func ColorCurRoundVotes() core.KVPainter {
	return wrapPainter(func(cns *CometNodeState) (string, interface{}) {
		if len(cns.Votes) <= cns.Round {
			return "round_votes", []int{}
		}
		votes := cns.Votes[cns.Round]
		prevotes := 0
		precommits := 0
		for _, pv := range votes.Prevotes {
			if pv != "nil-Vote" {
				prevotes += 1
			}
		}
		for _, pcv := range votes.Precommits {
			if pcv != "nil-Vote" {
				precommits += 1
			}
		}
		return "round_votes", []int{prevotes, precommits}
	})
}

func ColorVotes() core.KVPainter {
	return wrapPainter(func(s *CometNodeState) (string, interface{}) {
		bs, _ := json.Marshal(s.Votes)
		hash := sha256.Sum256(bs)
		return "votes", hex.EncodeToString(hash[:])
	})
}

func ColorProposer() core.KVPainter {
	return wrapPainter(func(s *CometNodeState) (string, interface{}) {
		proposerIndex := 0
		if s.Proposer != nil {
			proposerIndex = s.Proposer.Index
		}
		return "proposer", proposerIndex
	})
}
