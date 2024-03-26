package core

import (
	"errors"
	"math/rand"
	"sort"
	"time"

	"github.com/zeu5/dist-rl-testing/util"
)

var (
	ErrOutOfBounds = errors.New("state out of bounds")
)

type Message interface {
	Hash() string
	From() int
	To() int
}

type Request interface{}

type NState interface{}

type PState interface {
	NodeState(int) NState
	Messages() []Message
	Requests() []Request
	CanDeliverRequest() bool
}

type PEnvironment interface {
	Reset() (PState, error)
	Tick(*StepContext) (PState, error)
	DeliverMessages([]Message, *StepContext) (PState, error)
	DropMessages([]Message, *StepContext) (PState, error)
	ReceiveRequest(Request, *StepContext) (PState, error)
	StartNode(int, *StepContext) (PState, error)
	StopNode(int, *StepContext) (PState, error)
}

type Color interface {
	Hash() string
	Copy() Color
}

type InActiveColor struct {
	Color
}

func (i *InActiveColor) Hash() string {
	return "Inactive_" + i.Color.Hash()
}

func (i *InActiveColor) Copy() Color {
	return &InActiveColor{
		Color: i.Color.Copy(),
	}
}

// A painter that returns a color for the state
type Painter func(NState) Color

// A painter that returns key value for the state
// Should be used with ComposedPainter
type KVPainter func(NState) (string, interface{})

// ComposedPainter is a painter that is composed of multiple KVPainters
type ComposedPainter struct {
	SegPainters []KVPainter
}

// ComposedColor is a color that is composed of multiple colors
type ComposedColor struct {
	s map[string]interface{}
}

func (s *ComposedColor) Hash() string {
	return util.JsonHash(s.s)
}

func (s *ComposedColor) Copy() Color {
	newMap := make(map[string]interface{})
	for k, v := range s.s {
		newMap[k] = v
	}
	return &ComposedColor{s: newMap}
}

// NewComposedPainter returns a new ComposedPainter with the given KVPainters
func NewComposedPainter(sp ...KVPainter) *ComposedPainter {
	return &ComposedPainter{
		SegPainters: sp,
	}
}

// Painter returns a painter function that returns a ComposedColor
func (c *ComposedPainter) Painter() func(n NState) Color {
	return func(n NState) Color {
		s := make(map[string]interface{})
		for _, sp := range c.SegPainters {
			k, v := sp(n)
			s[k] = v
		}
		return &ComposedColor{s: s}
	}
}

type PEnvironmentConstructor interface {
	NewPEnvironment(int) PEnvironment
}

type PEnvironmentConfig struct {
	TicksBetweenPartition int
	Painter               Painter
	NumNodes              int
	MaxMessagesPerTick    int
	// Can stay in the same partition upto this number of ticks
	StaySameUpto    int
	WithCrashes     bool
	MaxCrashedNodes int
	MaxCrashActions int
	// Determines when a state is out of bounds
	BoundaryPredicate func(PState) bool
}

// partitionEnvironment is an environment that simulates a partitioned network
// The environment is composed of multiple nodes, each with a state and a color
// The environment can be in one of the partitions at any time
// The environment can change partitions based on the colors of the nodes
// The environment can also start and stop nodes
// The environment can deliver requests to the nodes
type partitionEnvironment struct {
	config *PEnvironmentConfig
	pEnv   PEnvironment

	rand     *rand.Rand
	curState *PartitionState
}

type PartitionState struct {
	NodeStates     map[int]NState
	messages       []Message
	Requests       []Request
	activeNodes    map[int]bool
	NodeColors     map[int]Color
	nodePartitions map[int]int
	repeatCount    int
	Partition      [][]Color

	numNodes          int
	canDeliverRequest bool
	withCrashes       bool
	remainingCrashes  int
	maxCrashedNodes   int
}

var _ State = &PartitionState{}

func (s *PartitionState) copy() *PartitionState {
	new := &PartitionState{
		NodeStates:     make(map[int]NState),
		messages:       make([]Message, 0),
		Requests:       make([]Request, 0),
		activeNodes:    make(map[int]bool),
		NodeColors:     make(map[int]Color),
		nodePartitions: make(map[int]int),
		repeatCount:    s.repeatCount,
		Partition:      make([][]Color, 0),

		numNodes:          s.numNodes,
		canDeliverRequest: s.canDeliverRequest,
		withCrashes:       s.withCrashes,
		remainingCrashes:  s.remainingCrashes,
		maxCrashedNodes:   s.maxCrashedNodes,
	}
	for i := 0; i < s.numNodes; i++ {
		new.NodeStates[i] = s.NodeStates[i]
	}
	new.messages = append(new.messages, s.messages...)
	new.Requests = append(new.Requests, s.Requests...)
	for i := range s.activeNodes {
		new.activeNodes[i] = true
	}
	for i := range s.nodePartitions {
		new.nodePartitions[i] = s.nodePartitions[i]
	}
	for i := range s.NodeColors {
		new.NodeColors[i] = s.NodeColors[i].Copy()
	}
	for i := range s.Partition {
		new.Partition = append(new.Partition, make([]Color, len(s.Partition[i])))
		for j := range s.Partition[i] {
			new.Partition[i][j] = s.Partition[i][j].Copy()
		}
	}
	return new
}

// Return a new state constructed using the given PState and the current state
func (s *PartitionState) copyWith(pState PState) *PartitionState {
	new := &PartitionState{
		NodeStates:     make(map[int]NState),
		messages:       make([]Message, 0),
		Requests:       make([]Request, 0),
		activeNodes:    make(map[int]bool),
		NodeColors:     make(map[int]Color),
		nodePartitions: make(map[int]int),
		repeatCount:    s.repeatCount,
		Partition:      make([][]Color, 0),

		numNodes:          s.numNodes,
		canDeliverRequest: pState.CanDeliverRequest(),
		withCrashes:       s.withCrashes,
		remainingCrashes:  s.remainingCrashes,
		maxCrashedNodes:   s.maxCrashedNodes,
	}
	for i := 0; i < s.numNodes; i++ {
		new.NodeStates[i] = pState.NodeState(i)
	}
	new.messages = append(new.messages, pState.Messages()...)
	new.Requests = append(new.Requests, pState.Requests()...)
	for i := range s.activeNodes {
		new.activeNodes[i] = true
	}
	for i := range s.nodePartitions {
		new.nodePartitions[i] = s.nodePartitions[i]
	}
	for i := 0; i < len(s.Partition); i++ {
		new.Partition = append(new.Partition, make([]Color, 0))
	}
	return new
}

// Paint the nodes based on the painter function
// Updates the partition based on the colors of the nodes (sorted lexicographically)
func (s *PartitionState) paint(painter Painter) {
	for i, ns := range s.NodeStates {
		c := painter(ns).Copy()
		if _, active := s.activeNodes[i]; active {
			s.NodeColors[i] = c
		} else {
			s.NodeColors[i] = &InActiveColor{Color: c}
		}
	}
	newPartition := make([][]Color, 0)
	for i := 0; i < len(s.Partition); i++ {
		newPartition = append(newPartition, make([]Color, 0))
	}
	for i, p := range s.nodePartitions {
		newPartition[p] = append(newPartition[p], s.NodeColors[i])
	}
	// Need to shrink empty partitions
	emptyPartitions := make([]int, 0)
	for i := 0; i < len(newPartition); i++ {
		if len(newPartition[i]) == 0 {
			emptyPartitions = append(emptyPartitions, i)
		}
	}
	// There can only be 1 empty partition
	// Empty partition occurs when we stop a node that is the only node in the partition
	// or when we start a node that is the only one inactive
	// We need to remove the empty partition
	// Update the partition map for all nodes whose partition index is greater than the empty partition index
	if len(emptyPartitions) > 0 {
		emptyPartition := emptyPartitions[0]
		newPartition = append(newPartition[:emptyPartition], newPartition[emptyPartition+1:]...)
		// Update the partition map
		for i, p := range s.nodePartitions {
			if p > emptyPartition {
				s.nodePartitions[i]--
			}
		}
	}

	for i, part := range newPartition {
		sort.Sort(colorSlice(part))
		newPartition[i] = part
	}
	sort.Sort(partitionSlice(newPartition))

	s.Partition = newPartition
}

// Compare the partition parameter between the two states
func (s *PartitionState) compare(other *PartitionState) bool {
	if len(s.Partition) != len(other.Partition) {
		return false
	}
	for i := 0; i < len(s.Partition); i++ {
		if len(s.Partition[i]) != len(other.Partition[i]) {
			return false
		}
		for j := 0; j < len(s.Partition[i]); j++ {
			if s.Partition[i][j].Hash() != other.Partition[i][j].Hash() {
				return false
			}
		}
	}
	return true
}

// Partition slice used to pass to the sort function
type partitionSlice [][]Color

func (p partitionSlice) Len() int { return len(p) }

// Two partitions are compared first based on the size and
// then lexicographically based on the colors
func (p partitionSlice) Less(i, j int) bool {
	if len(p[i]) < len(p[j]) {
		return true
	}
	less := false
	for k := 0; k < len(p[j]); k++ {
		one := p[i][k].Hash()
		two := p[j][k].Hash()
		if one < two {
			less = true
			break
		}
	}
	return less
}
func (p partitionSlice) Swap(i, j int) {
	p[i], p[j] = p[j], p[i]
}

// colorSlice used to pass to the sort interface
type colorSlice []Color

func (c colorSlice) Len() int { return len(c) }
func (c colorSlice) Less(i, j int) bool {
	one := c[i].Hash()
	two := c[j].Hash()
	return one < two
}
func (c colorSlice) Swap(i, j int) {
	c[i], c[j] = c[j], c[i]
}

var _ State = &PartitionState{}

// Hash returns the hash of the partition state
// The hash is computed based on the partition, repeat count and the number of requests
func (s *PartitionState) Hash() string {
	out := make(map[string]interface{})
	partition := make([][]string, 0)
	for i := range s.Partition {
		partition = append(partition, make([]string, 0))
		for j := range s.Partition[i] {
			partition[i] = append(partition[i], s.Partition[i][j].Hash())
		}
	}
	out["partition"] = partition
	out["repeat_count"] = s.repeatCount
	out["requests"] = len(s.Requests)
	return util.JsonHash(out)
}

// Actions returns the possible actions that can be taken from the current state
// 1. StaySamePartitionAction: Stay in the same partition
// 2. ChangePartitionAction: Change to a different partition based on current colors
// 3. StartAction: Start a node
// 4. StopAction: Stop a node
// 5. RequestAction: Deliver a request
func (s *PartitionState) Actions() []Action {
	colorElems := make([]util.Elem, 0)
	colorsHash := make(map[string]Color)
	for node, c := range s.NodeColors {
		hash := c.Hash()
		colorsHash[hash] = c
		if _, active := s.activeNodes[node]; active {
			colorElems = append(colorElems, util.StringElem(hash))
		}
	}
	actions := make([]Action, 0)
	if len(colorElems) > 0 {
		actions = append(actions, &StaySamePartitionAction{})
		for _, part := range util.EnumeratePartitions(util.MultiSet(colorElems)) {
			partition := make([][]Color, len(part))
			for i, ps := range part {
				partition[i] = make([]Color, len(ps))
				for j, c := range ps {
					hash := string(c.(util.StringElem))
					partition[i][j] = colorsHash[hash]
				}
			}
			inActivePart := make([]Color, 0)
			for n, c := range s.NodeColors {
				_, active := s.activeNodes[n]
				if !active {
					inActivePart = append(inActivePart, c)
				}
			}
			partition = append(partition, inActivePart)
			actions = append(actions, &ChangePartitionAction{Partition: partition})
		}
	}
	if s.withCrashes {
		totalNodes := s.numNodes
		crashedNodes := totalNodes - len(s.activeNodes)
		if (crashedNodes < s.maxCrashedNodes || s.maxCrashedNodes == 0) && s.remainingCrashes > 0 {

			activeColors := make(map[string]Color)
			inActiveColors := make(map[string]Color)
			for i := range s.NodeStates {
				if s.activeNodes[i] {
					activeColors[s.NodeColors[i].Hash()] = s.NodeColors[i]
				} else {
					inActiveColors[s.NodeColors[i].Hash()] = s.NodeColors[i]
				}
			}
			if len(activeColors) > 0 {
				for _, c := range activeColors {
					actions = append(actions, &StopAction{Color: c.Hash()})
				}
			}
			if len(inActiveColors) > 0 {
				actions = append(actions, &StartAction{})
			}
		}
	}
	if s.canDeliverRequest && len(s.Requests) > 0 {
		actions = append(actions, &RequestAction{})
	}
	return actions
}

type StartAction struct {
}

func (s *StartAction) Hash() string {
	return "start"
}

type StopAction struct {
	Color string
}

func (s *StopAction) Hash() string {
	return "stop-" + s.Color
}

type RequestAction struct {
	Request Request
}

func (r *RequestAction) Hash() string {
	return "request"
}

type StaySamePartitionAction struct {
}

func (s *StaySamePartitionAction) Hash() string {
	return "stay-same-partition"
}

type ChangePartitionAction struct {
	Partition [][]Color
}

func (c *ChangePartitionAction) Hash() string {
	out := "{ "
	for i := range c.Partition {
		out += " { "
		for j := range c.Partition[i] {
			out += c.Partition[i][j].Hash()
		}
		out += " } "
	}
	out += " }"
	return out
}

func (e *partitionEnvironment) Reset() (State, error) {
	pState, err := e.pEnv.Reset()
	if err != nil {
		return nil, err
	}
	newState := &PartitionState{
		NodeStates:     make(map[int]NState),
		messages:       pState.Messages(),
		Requests:       pState.Requests(),
		activeNodes:    make(map[int]bool),
		NodeColors:     make(map[int]Color),
		nodePartitions: make(map[int]int),
		repeatCount:    0,
		Partition:      make([][]Color, 0),

		numNodes:          e.config.NumNodes,
		canDeliverRequest: pState.CanDeliverRequest(),
		withCrashes:       e.config.WithCrashes,
		remainingCrashes:  e.config.MaxCrashActions,
		maxCrashedNodes:   e.config.MaxCrashedNodes,
	}
	colors := make([]Color, e.config.NumNodes)
	for i := 0; i < e.config.NumNodes; i++ {
		ns := pState.NodeState(i)
		color := e.config.Painter(ns)
		newState.activeNodes[i] = true
		newState.nodePartitions[i] = 0
		newState.NodeStates[i] = ns
		newState.NodeColors[i] = color
		colors[i] = color
	}
	sort.Sort(colorSlice(colors))
	newState.Partition = append(newState.Partition, colors)

	e.curState = newState.copy()
	return newState, nil
}

func (e *partitionEnvironment) Step(a Action, ctx *StepContext) (State, error) {
	switch act := a.(type) {
	case *StaySamePartitionAction:
		return e.handleStaySamePartition(ctx)
	case *ChangePartitionAction:
		return e.handleChangePartition(act, ctx)
	case *StartAction:
		return e.handleStartAction(act, ctx)
	case *StopAction:
		return e.handleStopAction(act, ctx)
	case *RequestAction:
		return e.handleRequestAction(ctx)
	default:
		return nil, errors.New("invalid action")
	}
}

func (e *partitionEnvironment) doTicks(ctx *StepContext) (*PartitionState, error) {
	var pState PState = nil
	var err error = nil
	for i := 0; i < e.config.TicksBetweenPartition; i++ {
		select {
		case <-ctx.Context.Done():
			return nil, ctx.Context.Err()
		default:
		}
		curMessages := make([]Message, 0)
		if pState == nil {
			curMessages = append(curMessages, e.curState.messages...)
		} else {
			curMessages = append(curMessages, pState.Messages()...)
		}
		if len(curMessages) > 0 {
			e.rand.Shuffle(len(curMessages), func(i, j int) {
				curMessages[i], curMessages[j] = curMessages[j], curMessages[i]
			})
			messagesToConsider := curMessages[:util.MinInt(len(curMessages), e.config.MaxMessagesPerTick)]
			toDeliver := make([]Message, 0)
			toDrop := make([]Message, 0)
			for _, m := range messagesToConsider {
				fromP := e.curState.nodePartitions[m.From()]
				toP := e.curState.nodePartitions[m.To()]
				_, toActive := e.curState.activeNodes[m.To()]
				if fromP == toP && toActive {
					toDeliver = append(toDeliver, m)
				} else if i == 0 {
					toDrop = append(toDrop, m)
				} else {
					curMessages = append(curMessages, m)
				}
			}
			if len(toDeliver) > 0 {
				_, err := e.pEnv.DeliverMessages(toDeliver, ctx)
				if err != nil {
					return nil, err
				}
			}
			if len(toDrop) > 0 {
				_, err := e.pEnv.DropMessages(toDrop, ctx)
				if err != nil {
					return nil, err
				}
			}
		}

		pState, err = e.pEnv.Tick(ctx)
		if err != nil {
			return nil, err
		}
	}

	if e.config.BoundaryPredicate != nil && e.config.BoundaryPredicate(pState) {
		return nil, ErrOutOfBounds
	}

	newState := e.curState.copyWith(pState)
	newState.paint(e.config.Painter)
	if e.curState.compare(newState) {
		if newState.repeatCount+1 < e.config.StaySameUpto {
			newState.repeatCount++
		}
	} else {
		newState.repeatCount = 0
	}

	select {
	case <-ctx.Context.Done():
		return nil, errors.New("context done")
	default:
	}

	return newState, nil
}

func (e *partitionEnvironment) handleStaySamePartition(ctx *StepContext) (State, error) {
	newState, err := e.doTicks(ctx)
	if err != nil {
		return nil, err
	}
	e.curState = newState
	return e.curState, nil
}

func (e *partitionEnvironment) handleChangePartition(act *ChangePartitionAction, ctx *StepContext) (State, error) {
	newPartitionMap := make(map[int]int)
	colorNodes := make(map[string][]int)
	for i, c := range e.curState.NodeColors {
		cHash := c.Hash()
		if _, ok := colorNodes[cHash]; !ok {
			colorNodes[cHash] = make([]int, 0)
		}
		colorNodes[cHash] = append(colorNodes[cHash], i)
	}
	newPartition := make([][]Color, 0)
	for i, part := range act.Partition {
		partition := make([]Color, 0)
		for _, c := range part {
			cHash := c.Hash()
			nextNode := colorNodes[cHash][0]
			colorNodes[cHash] = colorNodes[cHash][1:]
			newPartitionMap[nextNode] = i
			partition = append(partition, c)
		}
		sort.Sort(colorSlice(partition))
		newPartition = append(newPartition, partition)
	}
	sort.Sort(partitionSlice(newPartition))
	newState := e.curState.copy()
	newState.nodePartitions = newPartitionMap
	newState.Partition = newPartition
	e.curState = newState
	newState, err := e.doTicks(ctx)
	if err != nil {
		return nil, err
	}
	e.curState = newState
	return e.curState, nil
}

func (e *partitionEnvironment) handleStartAction(_ *StartAction, ctx *StepContext) (State, error) {
	newState := e.curState.copy()
	inactive := make([]int, 0)
	for i := 0; i < e.config.NumNodes; i++ {
		if _, active := newState.activeNodes[i]; !active {
			inactive = append(inactive, i)
		}
	}
	var toActivate int = -1
	if len(inactive) > 0 {
		toActivate = inactive[e.rand.Intn(len(inactive))]
		newState.activeNodes[toActivate] = true

		_, err := e.pEnv.StartNode(toActivate, ctx)
		if err != nil {
			return nil, err
		}
	}
	pState, err := e.pEnv.Tick(ctx)
	if err != nil {
		return nil, err
	}
	if e.config.BoundaryPredicate != nil && e.config.BoundaryPredicate(pState) {
		return nil, ErrOutOfBounds
	}
	// Add new partition for the newly started node
	newState = newState.copyWith(pState)
	if toActivate != -1 {
		newState.nodePartitions[toActivate] = len(newState.Partition)
		newState.Partition = append(newState.Partition, make([]Color, 0))
	}

	newState.paint(e.config.Painter)
	if e.curState.compare(newState) {
		if newState.repeatCount+1 < e.config.StaySameUpto {
			newState.repeatCount++
		}
	} else {
		newState.repeatCount = 0
	}
	e.curState = newState
	return e.curState, nil
}

func (e *partitionEnvironment) handleStopAction(act *StopAction, ctx *StepContext) (State, error) {
	newState := e.curState.copy()
	newState.remainingCrashes--
	activeNodes := make([]int, 0)
	for node, c := range newState.NodeColors {
		if c.Hash() == act.Color {
			activeNodes = append(activeNodes, node)
		}
	}
	var toDeactivate int = -1
	if len(activeNodes) > 0 {
		toDeactivate = activeNodes[e.rand.Intn(len(activeNodes))]
		delete(newState.activeNodes, toDeactivate)
		_, err := e.pEnv.StopNode(toDeactivate, ctx)
		if err != nil {
			return nil, err
		}
	}
	pState, err := e.pEnv.Tick(ctx)
	if err != nil {
		return nil, err
	}
	if toDeactivate != -1 {
		// We have decided to deactivate a node
		haveOtherInActive := e.config.NumNodes-len(e.curState.activeNodes) > 1
		if haveOtherInActive {
			// There is already a partition with inactive nodes which we can use
			inActivePartition := -1
			for i := 0; i < e.config.NumNodes; i++ {
				if _, active := e.curState.activeNodes[i]; !active {
					inActivePartition = e.curState.nodePartitions[i]
					break
				}
			}
			newState.nodePartitions[toDeactivate] = inActivePartition
		} else {
			// We need to create a new partition for the inactive node
			newState.nodePartitions[toDeactivate] = len(newState.Partition)
			newState.Partition = append(newState.Partition, make([]Color, 0))
		}
	}
	if e.config.BoundaryPredicate != nil && e.config.BoundaryPredicate(pState) {
		return nil, ErrOutOfBounds
	}
	newState = newState.copyWith(pState)
	newState.paint(e.config.Painter)
	if e.curState.compare(newState) {
		if newState.repeatCount+1 < e.config.StaySameUpto {
			newState.repeatCount++
		}
	} else {
		newState.repeatCount = 0
	}
	e.curState = newState
	return e.curState, nil
}

func (e *partitionEnvironment) handleRequestAction(ctx *StepContext) (State, error) {
	req := e.curState.Requests[0]
	newState := e.curState.copy()
	newState.Requests = newState.Requests[1:]
	_, err := e.pEnv.ReceiveRequest(req, ctx)
	if err != nil {
		return nil, err
	}
	pState, err := e.pEnv.Tick(ctx)
	if err != nil {
		return nil, err
	}
	if e.config.BoundaryPredicate != nil && e.config.BoundaryPredicate(pState) {
		return nil, ErrOutOfBounds
	}
	newState = newState.copyWith(pState)
	newState.paint(e.config.Painter)
	if e.curState.compare(newState) {
		if newState.repeatCount+1 < e.config.StaySameUpto {
			newState.repeatCount++
		}
	} else {
		newState.repeatCount = 0
	}
	e.curState = newState
	return e.curState, nil
}

var _ EnvironmentConstructor = &partitionEnvironmentConstructor{}

// partitionEnvironmentConstructor is an environment constructor for the partition environment
// Separates the config and the PEnvironmentConstructor
type partitionEnvironmentConstructor struct {
	PEnvironmentConstructor PEnvironmentConstructor
	config                  *PEnvironmentConfig
}

func defaultPartitionState(config *PEnvironmentConfig) *PartitionState {
	out := &PartitionState{
		NodeStates:     make(map[int]NState),
		messages:       make([]Message, 0),
		Requests:       make([]Request, 0),
		activeNodes:    make(map[int]bool),
		NodeColors:     make(map[int]Color),
		nodePartitions: make(map[int]int),
		repeatCount:    0,
		Partition:      make([][]Color, 0),

		numNodes:          config.NumNodes,
		canDeliverRequest: false,
		withCrashes:       config.WithCrashes,
		remainingCrashes:  config.MaxCrashActions,
		maxCrashedNodes:   config.MaxCrashedNodes,
	}
	for i := 0; i < config.NumNodes; i++ {
		out.activeNodes[i] = true
		out.nodePartitions[i] = 0
	}
	out.Partition = append(out.Partition, make([]Color, 0))
	return out
}

func (pc *partitionEnvironmentConstructor) NewEnvironment(instance int) Environment {

	return &partitionEnvironment{
		config: pc.config,
		pEnv:   pc.PEnvironmentConstructor.NewPEnvironment(instance),

		rand:     rand.New(rand.NewSource(time.Now().UnixNano())),
		curState: defaultPartitionState(pc.config),
	}
}

// GetConstructor returns a new EnvironmentConstructor based on the given PEnvironmentConstructor
func (c *PEnvironmentConfig) GetConstructor(pec PEnvironmentConstructor) EnvironmentConstructor {
	return &partitionEnvironmentConstructor{
		PEnvironmentConstructor: pec,
		config:                  c,
	}
}
