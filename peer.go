package scr

import (
	"math/rand"
)

type PeerList interface {
	Locations() []V // For visualization only
	RemovePeer(o *Node)
	AddPeer(loc V, o *Node) bool
	GetRandomPeer() *Node
	GetRandomPeerThatsNot(o *Node) *Node
	RandomlyFindPeerCloserToData(loc V, data []*Data, indices []int) (peer *Node, idx int)
	IterateOverPeersWith(func(*Node))
}

var _ PeerList = &basePeerList{}

type basePeerList struct {
	uniqueIdx     map[*Node]int
	peers         []*Node
	peerLocations []V
	maxPeers      int
}

func newBasePeerList(n int) *basePeerList {
	return &basePeerList{
		uniqueIdx:     make(map[*Node]int, n),
		peers:         make([]*Node, 0, n),
		peerLocations: make([]V, 0, n),
		maxPeers:      n,
	}
}

func (p *basePeerList) Locations() []V {
	return p.peerLocations
}

func (p *basePeerList) RemovePeer(o *Node) {
	if i, ok := p.uniqueIdx[o]; ok {
		p.peers[i] = p.peers[len(p.peers)-1]
		p.peers[len(p.peers)-1] = nil
		p.peers = p.peers[:len(p.peers)-1]
		p.peerLocations[i] = p.peerLocations[len(p.peerLocations)-1]
		p.peerLocations[len(p.peerLocations)-1] = V{}
		p.peerLocations = p.peerLocations[:len(p.peerLocations)-1]
		delete(p.uniqueIdx, o)
	}
}

func (p *basePeerList) AddPeer(loc V, o *Node) bool {
	if i, ok := p.uniqueIdx[o]; ok {
		p.peerLocations[i] = o.Location
		return true
	}
	if len(p.peers) < p.maxPeers {
		p.uniqueIdx[o] = len(p.peers)
		p.peers = append(p.peers, o)
		p.peerLocations = append(p.peerLocations, o.Location)
		return true
	}
	return false
}

func (p *basePeerList) GetRandomPeer() *Node {
	if len(p.peers) == 0 {
		return nil
	}
	return p.peers[rand.Intn(len(p.peers))]
}

func (p *basePeerList) GetRandomPeerThatsNot(o *Node) *Node {
	for i := 0; i < getPeerForMaxTries; i++ {
		idx := rand.Intn(len(p.peers))
		if p := p.peers[idx]; p != o {
			return p
		}
	}
	return nil
}

func (p *basePeerList) length() int {
	return len(p.peers)
}

func (p *basePeerList) RandomlyFindPeerCloserToData(loc V, data []*Data, indices []int) (peer *Node, idx int) {
	if len(p.peers) == 0 {
		return nil, -1
	}
	// Randomly begin asking peers
	offset := rand.Intn(len(p.peers))
	quit := false
	i := offset
	dataIdx := -1
	for !quit {
		// See if they're closer to an address.
		for _, d := range indices {
			data := data[d]
			if data == nil {
				continue
			}
			if loc.GreatCircleDistance(data.Location) > p.peerLocations[i].GreatCircleDistance(data.Location) {
				dataIdx = d
				quit = true
				break
			}
		}
		if quit { // Hack: don't increment i
			break
		}
		// Wrap around when searching peers.
		i++
		if i >= len(p.peers) {
			i = 0
		}
		if i == offset {
			quit = true
		}
	}
	return p.peers[i], dataIdx
}

func (p *basePeerList) IterateOverPeersWith(f func(*Node)) {
	for _, n := range p.peers {
		f(n)
	}
}

var _ PeerList = &maximizePeerSpread{}

type maximizePeerSpread struct {
	*basePeerList
}

func NewMaximizePeerSpread(n int) PeerList {
	return &maximizePeerSpread{
		newBasePeerList(n),
	}
}

func (p *maximizePeerSpread) AddPeer(loc V, o *Node) bool {
	if i, ok := p.uniqueIdx[o]; ok {
		p.peerLocations[i] = o.Location
		return true
	}
	if len(p.peers) < p.maxPeers {
		p.uniqueIdx[o] = len(p.peers)
		p.peers = append(p.peers, o)
		p.peerLocations = append(p.peerLocations, o.Location)
		return true
	} else {
		// eliminate lowest distance algorithm
		dists := make([]float64, len(p.peers))
		distO := 0.0
		distOs := make([]float64, len(p.peers))
		for i := 0; i < len(p.peers); i++ {
			dist := p.peerLocations[i].GreatCircleDistance(o.Location)
			distO += dist
			distOs[i] = dist
			for j := 0; j < i; j++ {
				dist = p.peerLocations[i].GreatCircleDistance(p.peerLocations[j])
				dists[i] += dist
				dists[j] += dist
			}
		}
		idx := -1
		for i, dist := range dists {
			if dist < distO-distOs[i] {
				if idx == -1 || dist < dists[idx] {
					idx = i
				}
			}
		}
		if idx >= 0 {
			delete(p.uniqueIdx, p.peers[idx])
			p.uniqueIdx[o] = idx
			p.peers[idx] = o
			p.peerLocations[idx] = o.Location
			return true
		}
	}
	return false
}

var _ PeerList = &closestNeighbors{}

type closestNeighbors struct {
	*basePeerList
}

func NewClosestNeighbors(n int) PeerList {
	return &closestNeighbors{
		newBasePeerList(n),
	}
}

func (p *closestNeighbors) AddPeer(loc V, o *Node) bool {
	if i, ok := p.uniqueIdx[o]; ok {
		p.peerLocations[i] = o.Location
		return true
	}
	if len(p.peers) < p.maxPeers {
		p.uniqueIdx[o] = len(p.peers)
		p.peers = append(p.peers, o)
		p.peerLocations = append(p.peerLocations, o.Location)
		return true
	} else {
		dists := make([]float64, len(p.peers))
		distO := loc.GreatCircleDistance(o.Location)
		for i := 0; i < len(p.peers); i++ {
			dist := loc.GreatCircleDistance(p.peerLocations[i])
			dists[i] = dist
		}
		maxO := true
		idx := -1
		maxDist := distO
		for i, dist := range dists {
			if dist > maxDist {
				maxO = false
				idx = i
			}
		}
		if !maxO {
			delete(p.uniqueIdx, p.peers[idx])
			p.uniqueIdx[o] = idx
			p.peers[idx] = o
			p.peerLocations[idx] = o.Location
			return true
		}
	}
	return false
}

var _ PeerList = &maxSpreadThenClosestNeighbors{}

type maxSpreadThenClosestNeighbors struct {
	M *maximizePeerSpread
	C *closestNeighbors
}

func NewMaxSpreadThenClosestNeighbors(nm, nc int) PeerList {
	return &maxSpreadThenClosestNeighbors{
		M: &maximizePeerSpread{
			newBasePeerList(nm),
		},
		C: &closestNeighbors{
			newBasePeerList(nc),
		},
	}
}

func (p *maxSpreadThenClosestNeighbors) Locations() []V {
	return append(p.M.Locations(), p.C.Locations()...)
}

func (p *maxSpreadThenClosestNeighbors) RemovePeer(o *Node) {
	p.M.RemovePeer(o)
	p.C.RemovePeer(o)
}

func (p *maxSpreadThenClosestNeighbors) AddPeer(loc V, o *Node) bool {
	ok := p.M.AddPeer(loc, o)
	if ok {
		return ok
	}
	ok = p.C.AddPeer(loc, o)
	return ok
}

func (p *maxSpreadThenClosestNeighbors) GetRandomPeer() *Node {
	if p.M.length() > 0 && p.C.length() > 0 {
		i := rand.Intn(2)
		if i == 0 {
			return p.M.GetRandomPeer()
		} else {
			return p.C.GetRandomPeer()
		}
	} else if p.M.length() > 0 {
		return p.M.GetRandomPeer()
	}
	return p.C.GetRandomPeer()
}

func (p *maxSpreadThenClosestNeighbors) GetRandomPeerThatsNot(o *Node) *Node {
	if p.M.length() > 0 && p.C.length() > 0 {
		i := rand.Intn(2)
		if i == 0 {
			return p.M.GetRandomPeerThatsNot(o)
		} else {
			return p.C.GetRandomPeerThatsNot(o)
		}
	} else if p.M.length() > 0 {
		return p.M.GetRandomPeerThatsNot(o)
	}
	return p.C.GetRandomPeerThatsNot(o)
}

func (p *maxSpreadThenClosestNeighbors) RandomlyFindPeerCloserToData(loc V, data []*Data, indices []int) (peer *Node, idx int) {
	if p.M.length() > 0 && p.C.length() > 0 {
		peer, idx = p.M.RandomlyFindPeerCloserToData(loc, data, indices)
		if idx < 0 {
			peer, idx = p.C.RandomlyFindPeerCloserToData(loc, data, indices)
		}
	} else if p.M.length() > 0 {
		peer, idx = p.M.RandomlyFindPeerCloserToData(loc, data, indices)
	} else {
		peer, idx = p.C.RandomlyFindPeerCloserToData(loc, data, indices)
	}
	return
}

func (p *maxSpreadThenClosestNeighbors) IterateOverPeersWith(f func(*Node)) {
	p.M.IterateOverPeersWith(f)
	p.C.IterateOverPeersWith(f)
}
