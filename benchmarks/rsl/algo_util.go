package rsl

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
)

type Entry struct {
	Accepted bool
	Ballot   Ballot
	Decree   int
	Command  Command
}

func (e Entry) Copy() Entry {
	return Entry{
		Accepted: e.Accepted,
		Ballot:   e.Ballot.Copy(),
		Decree:   e.Decree,
		Command:  e.Command.Copy(),
	}
}

type Log struct {
	entries []Entry
	Decided []Command
}

func NewLog() *Log {
	return &Log{
		entries: make([]Entry, 0),
		Decided: make([]Command, 0),
	}
}

func (l *Log) Add(e Entry) {
	l.entries = append(l.entries, e)
}

func (l *Log) AddDecided(c Command) {
	l.Decided = append(l.Decided, c.Copy())
}

func (l *Log) NumDecided() int {
	return len(l.Decided)
}

func (l *Log) NumEntries() int {
	return len(l.entries)
}

func (l *Log) Copy() *Log {
	out := NewLog()
	for _, e := range l.entries {
		out.Add(e.Copy())
	}
	for _, c := range l.Decided {
		out.AddDecided(c.Copy())
	}
	return out
}

type MessageType string

var (
	MessageAcceptReq   MessageType = "AcceptReq"
	MessageAcceptResp  MessageType = "AcceptResp"
	MessageStatusReq   MessageType = "StatusReq"
	MessageStatusResp  MessageType = "StatusResp"
	MessagePrepareReq  MessageType = "PrepareReq"
	MessagePrepareResp MessageType = "PrepareResp"
	MessageFetchReq    MessageType = "FetchReq"
	MessageFetchResp   MessageType = "FetchResp"
	MessageCommand     MessageType = "CommandMessage"
)

type Message struct {
	From           uint64
	To             uint64
	Type           MessageType
	Ballot         Ballot
	Decree         int
	Command        Command
	ProposalBallot Ballot
	Timestamp      int
	Accept         bool
	Proposals      []Proposal
}

func (m Message) Hash() string {
	bs, _ := json.Marshal(m)
	hash := sha256.Sum256(bs)
	return hex.EncodeToString(hash[:])
}

func (m Message) Copy() Message {
	new := Message{
		From:   m.From,
		To:     m.To,
		Type:   m.Type,
		Ballot: m.Ballot.Copy(),
		Decree: m.Decree,
		Command: Command{
			Data: bytes.Clone(m.Command.Data),
		},
		ProposalBallot: m.ProposalBallot.Copy(),
		Timestamp:      m.Timestamp,
		Accept:         m.Accept,
		Proposals:      make([]Proposal, len(m.Proposals)),
	}
	for i, p := range m.Proposals {
		new.Proposals[i] = p.Copy()
	}
	return new
}

func (n *Node) broadcast(m Message) {
	m.From = n.ID
	for _, p := range n.config.Peers {
		if p == n.ID {
			continue
		}
		mC := m.Copy()
		mC.To = p
		n.pendingMessagesToSend = append(n.pendingMessagesToSend, mC)
	}
}

func (n *Node) send(m Message) {
	m.From = n.ID
	n.pendingMessagesToSend = append(n.pendingMessagesToSend, m.Copy())
}

// Configuration of a RSL node, contains the set of peers
type RSLConfig struct {
	Number        int
	InitialDecree int
	Members       map[uint64]bool
}

func (r RSLConfig) QuorumSize() int {
	return (len(r.Members))/2 + 1
}

func (r RSLConfig) Copy() RSLConfig {
	n := RSLConfig{
		Number:        r.Number,
		InitialDecree: r.InitialDecree,
		Members:       make(map[uint64]bool),
	}
	for k := range r.Members {
		n.Members[k] = true
	}
	return n
}

type RSLConfigCommand struct {
	Members map[uint64]bool
}

func NewRSLConfigCommand(peers []uint64) *RSLConfigCommand {
	c := &RSLConfigCommand{
		Members: make(map[uint64]bool),
	}
	for _, p := range peers {
		c.Members[p] = true
	}
	return c
}

func (r *RSLConfigCommand) Add(peer uint64) {
	r.Members[peer] = true
}

func (r *RSLConfigCommand) Remove(peer uint64) {
	delete(r.Members, peer)
}

func (r *RSLConfigCommand) ToCommand() Command {
	c := Command{}
	bs, _ := json.Marshal(r)
	c.Data = bytes.Clone(bs)
	return c
}

func IsConfigCommand(c Command) bool {
	cfg := RSLConfigCommand{}
	err := json.Unmarshal(c.Data, &cfg)
	return err == nil
}

func GetRSLConfig(c Command) (*RSLConfigCommand, bool) {
	cfg := &RSLConfigCommand{}
	err := json.Unmarshal(c.Data, cfg)
	if err != nil {
		return nil, false
	}
	return cfg, true
}
