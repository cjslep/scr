package scr

import (
	"fmt"
	"math/rand"
)

const (
	getPeerForMaxTries = 10
)

// AKA a floating castle in the sky
//
// There's a place in my mind
// No one knows where it hides
// And my fantasy is flying
// It's a castle in the sky
type Node struct {
	Location     V
	S            State
	NextS        State
	CurrentBSize int
	MaxBSize     int
	WaitActivity float64
	// Data is the slice into the global slice, for rapid lookup. Presence
	// of Data does not indicate memory ownership.
	Data []*Data
	// DataIndices are a slice of indices into the Data slice that this node
	// owns. It is preallocated to a fixed size, indicating a numerical
	// limit the node can hold. The content of the data it holds can be
	// accessed by:
	//
	//     Data[DataIndices[0:len(DataIndices)-1]]
	//
	// If the resulting value is nil, then it is empty
	DataIndices []int
	// This node's known peers
	peers PeerList
	// The f(X) value for this node (lower = closer to its data)
	fx float64
	// Sum sum of the square f(X) value (for std dev calculations)
	fxsq float64
	// Number of data pieces that go into the f(X) calculation
	nfx int
}

// NewNodes begin at a random location if they have no data. Otherwise, they
// begin at a predetermined location based on the data they possess.
func NewNode(
	dataCache []*Data,
	myIdxs []int,
	maxBSize int,
	waitActivity float64,
	peerList PeerList) *Node {
	n := &Node{
		S:            State{id: StateJoin},
		Data:         dataCache,
		DataIndices:  myIdxs,
		MaxBSize:     maxBSize,
		WaitActivity: waitActivity,
		peers:        peerList,
	}
	n.computeLocationAndCurrentSize()
	return n
}

func (n *Node) getDataLocations() []V {
	locs := make([]V, 0, len(n.DataIndices))
	for _, idx := range n.DataIndices {
		if n.Data[idx] == nil {
			continue
		}
		locs = append(locs, n.Data[idx].Location)
	}
	return locs
}

func (n *Node) computeLocationAndCurrentSize() {
	bsz := 0
	locs := make([]V, 0, len(n.DataIndices))
	weights := make([]float64, 0, len(n.DataIndices))
	hasData := false
	for _, idx := range n.DataIndices {
		if n.Data[idx] == nil {
			continue
		}
		hasData = true
		locs = append(locs, n.Data[idx].Location)
		weights = append(weights, 1)
		bsz += n.Data[idx].DataSize
	}
	if hasData {
		var err error
		var fx float64
		var fxsq float64
		var nfx int
		n.Location, fx, fxsq, nfx, err = SolveNonEuclideanMultifacilityLocationMonteCarlo(
			locs,
			weights,
			0.1, 0.1,
			2)
		if err != nil {
			// TODO: Yikes!
			panic(err)
		}
		n.fx = fx
		n.fxsq = fxsq
		n.nfx = nfx
	} else {
		fmt.Printf("RandomVector location: nIdx=%v\n", len(n.DataIndices))
		n.Location = RandomVector()
		n.fx = 0
		n.fxsq = 0
		n.nfx = 0
	}
	n.CurrentBSize = bsz
}

func (n *Node) ifWaitOrJoin(f func()) bool {
	if n.S.id == StateWait || n.S.id == StateJoin {
		f()
		return true
	}
	return false
}

func (n *Node) ifWaitOrJoinBool(f func() bool) (executed bool, val bool) {
	if n.S.id == StateWait || n.S.id == StateJoin {
		return true, f()
	}
	return false, false
}

func (n *Node) addPeer(o *Node) {
	n.peers.AddPeer(n.Location, o)
}

func (n *Node) requestPeer() {
	o := n.peers.GetRandomPeer()
	if o == nil {
		return
	}
	peer := o.getPeerFor(n)
	if peer != nil {
		// NODE INTERACTION: PEER HELLO
		if o.ifWaitOrJoin(func() { o.addPeer(n) }) {
			n.addPeer(peer)
		}
	}
}

func (n *Node) getPeerFor(o *Node) *Node {
	return n.peers.GetRandomPeerThatsNot(o)
}

func (n *Node) exchangeData() (s string) {
	o, dataIdx := n.peers.RandomlyFindPeerCloserToData(n.Location, n.Data, n.DataIndices)
	if o == nil && dataIdx < 0 {
		s = "could not exchange data (no peers)"
		return
	}
	// dataIdx is data to transfer if >= 0; otherwise return
	// if peer says "ok" then we forget our reference
	if dataIdx < 0 {
		s = "could not exchange data (no closer data)"
		return
	}
	exec, ok := o.ifWaitOrJoinBool(func() bool { return o.exchangeDataReceive(n.Data[dataIdx]) })
	if !exec {
		s = fmt.Sprintf("unsuccessfully exchanged data at index %d to peer %v", dataIdx, o.Location)
		return
	}
	if ok {
		n.exchangeDataGive(dataIdx)
		s = fmt.Sprintf("exchanged data at index %d to peer %v", dataIdx, o.Location)
	}
	// Exchange locations after data exchange
	o.addPeer(n) // Not wrapped since exec succeeded before
	n.addPeer(o)
	return
}

func (n *Node) exchangeDataReceive(d *Data) bool {
	// Too big of data
	if d.DataSize+n.CurrentBSize > n.MaxBSize {
		return false
	}
	availIdx := -1
	for _, idx := range n.DataIndices {
		if n.Data[idx] == nil {
			availIdx = idx
			break
		}
	}
	// Too much qty of data
	if availIdx < 0 {
		return false
	}
	n.Data[availIdx] = d
	n.computeLocationAndCurrentSize()
	return true
}

func (n *Node) exchangeDataGive(idx int) {
	n.Data[idx] = nil
	n.computeLocationAndCurrentSize()
}

func (n *Node) nextFreeDataIndex() int {
	availIdx := -1
	for _, idx := range n.DataIndices {
		if n.Data[idx] == nil {
			availIdx = idx
			break
		}
	}
	return availIdx
}

func (n *Node) applyNewData(idx int) {
	d := n.Data[idx]
	// Too big of data -- remove it
	if d.DataSize+n.CurrentBSize > n.MaxBSize {
		n.Data[idx] = nil
	}
	n.computeLocationAndCurrentSize()
}

type coordinator interface {
	FindOtherArbitraryNode(*Node) *Node
}

func (n *Node) ApplyState(c coordinator) string {
	switch n.S.id {
	case StateJoin:
		o := c.FindOtherArbitraryNode(n)
		s := "did not find other arbitrary node"
		if o != nil {
			// Exchange location information as well.
			// May not be a mutual add.
			//
			// NODE INTERACTION: PEER HELLO
			if o.ifWaitOrJoin(func() { o.addPeer(n) }) {
				s = "found other arbitrary node"
				n.addPeer(o)
			}
		}
		n.NextS = State{
			id:        StateWait,
			lastState: StateJoin,
		}
		return fmt.Sprintf("Node at %s joined and %s", n.Location, s)
	case StateWait:
		// Chance of the node spontaneously doing an action. We simply
		// attempt to alternate through actions.
		if rand.Float64() < n.WaitActivity {
			if n.S.lastState == StateExchangeData {
				n.NextS = State{
					id:        StateAskPeer,
					lastState: StateWait,
				}
				return fmt.Sprintf("Node at %s will attempt asking peer", n.Location)
			} else {
				n.NextS = State{
					id:        StateExchangeData,
					lastState: StateWait,
				}
				return fmt.Sprintf("Node at %s will attempt exchange", n.Location)
			}
		} else {
			n.NextS = State{
				id:        StateWait,
				lastState: StateWait,
			}
			return fmt.Sprintf("Node at %s waited", n.Location)
		}
	case StateExchangeData:
		// Try exchanging data with a neighbor closer
		// to that data's location
		//
		// NODE INTERACTION: EXCHANGE DATA
		s := n.exchangeData()
		n.NextS = State{
			id:        StateWait,
			lastState: StateExchangeData,
		}
		return fmt.Sprintf("Node at %s %s", n.Location, s)
	case StateAskPeer:
		// Try asking for a peer
		//
		// NODE INTERACTION: REQUEST PEER
		n.requestPeer()
		n.NextS = State{
			id:        StateWait,
			lastState: StateAskPeer,
		}
		return fmt.Sprintf("Node at %s asked peer", n.Location)
	}
	return fmt.Sprintf("Unknown action: %v", n.S)
}

func (n *Node) AdvanceState() {
	n.S = n.NextS
}

func (n *Node) CountLastState(m []int) {
	m[n.NextS.lastState] += 1
}

func (n *Node) PeerLocations() []V {
	return n.peers.Locations()
}

// removePeer is used by Tocker via Simulation
func (n *Node) removePeer(o *Node) {
	n.peers.RemovePeer(o)
}

type State struct {
	id        int
	lastState int
}

const (
	StateJoin int = iota
	StateWait
	StateExchangeData
	StateAskPeer
)
