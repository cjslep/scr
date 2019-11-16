package scr

import (
	"fmt"
	"math"
	"math/rand"
	"os"
	"sync"
	"time"
)

const (
	nTickMilli = 16
)

// isHopHistIter determines if an iteration is interesting enough to
// pause and do expensive hop histogram / disjointedness calculations.
func isHopHistIter(i int) bool {
	return i == 0 ||
		i == 1 ||
		i == 50 ||
		i == 5001 ||
		i == 5050 ||
		i%1000 == 0
}

type Tocker interface {
	// Given a simulation and iteration, may modify the simulation.
	Tock(s *Simulation, i int)
}

type CreateDataFactoryFn func() CreateDataFn
type CreateDataFn func() []byte
type AllocateNDataToNodeFactoryFn func() func(nFree int) int
type NodeInitialDataFactoryFn func() func(nSize int) int
type NodeMaxBSizeFactoryFn func() func(currSize int) int
type WaitActivityFactoryFn func() func() float64
type DataGrowthChanceFactoryFn func() func() float64
type PeerListFactoryFn func() func() PeerList

type Simulation struct {
	DataCache        []*Data // Preallocated, global slice
	DataAllocdToNode []bool  // Same len as DataCache
	NDataCacheFree   int     // Number of DataAllocd To Node with "false" value.
	NodeCache        []*Node // Preallocated size for the lifetime of the simulation
	Tockers          []Tocker

	NodeInitialDataFactoryFn     NodeInitialDataFactoryFn
	AllocateNDataToNodeFactoryFn AllocateNDataToNodeFactoryFn
	CreateDataFactoryFn          CreateDataFactoryFn
	NodeMaxBSizeFactoryFn        NodeMaxBSizeFactoryFn
	WaitActivityFactoryFn        WaitActivityFactoryFn
	DataGrowthChanceFactoryFn    DataGrowthChanceFactoryFn
	PeerListFactoryFn            PeerListFactoryFn

	TickN         int
	Log           *os.File
	NodeFile      *os.File
	NodeStateFile *os.File
	FxFile        *os.File
	vizOnly       bool
	doneCh        chan bool
	ackDoneCh     chan bool
	pauseCh       chan bool
	playCh        chan bool
	mu            *sync.RWMutex

	redraw func(i, fx, nfx int, avg, stddev float64, dur, durLockless time.Duration)
}

func NewSimulation(
	nStartNodes int,
	nMaxData int,
	nMaxNode int,
	tockers []Tocker,
	nodeInitialDataFactoryFn NodeInitialDataFactoryFn,
	allocateNDataToNodeFactoryFn AllocateNDataToNodeFactoryFn,
	createDataFactoryFn CreateDataFactoryFn,
	nodeMaxBSizeFactoryFn NodeMaxBSizeFactoryFn,
	waitActivityFactoryFn WaitActivityFactoryFn,
	dataGrowthChanceFactoryFn DataGrowthChanceFactoryFn,
	peerListFactoryFn PeerListFactoryFn,
	vizOnly bool) *Simulation {
	s := &Simulation{
		DataCache:                    make([]*Data, nMaxData),
		DataAllocdToNode:             make([]bool, nMaxData),
		NDataCacheFree:               nMaxData,
		NodeCache:                    make([]*Node, nMaxNode),
		Tockers:                      tockers,
		NodeInitialDataFactoryFn:     nodeInitialDataFactoryFn,
		AllocateNDataToNodeFactoryFn: allocateNDataToNodeFactoryFn,
		CreateDataFactoryFn:          createDataFactoryFn,
		NodeMaxBSizeFactoryFn:        nodeMaxBSizeFactoryFn,
		WaitActivityFactoryFn:        waitActivityFactoryFn,
		DataGrowthChanceFactoryFn:    dataGrowthChanceFactoryFn,
		PeerListFactoryFn:            peerListFactoryFn,
		TickN:                        0,
		vizOnly:                      vizOnly,
		doneCh:                       make(chan bool),
		ackDoneCh:                    make(chan bool),
		pauseCh:                      make(chan bool),
		playCh:                       make(chan bool),
		mu:                           &sync.RWMutex{},
	}
	for i := 0; i < nStartNodes && i < len(s.NodeCache); i++ {
		s.NodeCache[i] = s.createNode()
	}
	return s
}

func (s *Simulation) SetRedraw(f func(i, fx, nfx int, avg, stddev float64, dur, durLockless time.Duration)) {
	s.redraw = f
}

// NewNodeJoins is for Tockers to use
func (s *Simulation) NewNodeJoins() {
	for i := 0; i < len(s.NodeCache); i++ {
		if s.NodeCache[i] != nil {
			continue
		}
		s.NodeCache[i] = s.createNode()
		return
	}
}

// ExistingNodeLeaves is for Tockers to use
func (s *Simulation) ExistingNodeLeaves() {
	var d *Node
	for i := len(s.NodeCache) - 1; i >= 0; i-- {
		if s.NodeCache[i] == nil {
			continue
		}
		if d == nil {
			d = s.NodeCache[i]
			s.NodeCache[i] = nil
		} else {
			s.NodeCache[i].removePeer(d)
		}
	}
}

// GenerateLocalData is for Tockers to use
func (s *Simulation) GenerateLocalData() {
	chanceFn := s.DataGrowthChanceFactoryFn()
	createDataFn := s.CreateDataFactoryFn()
	for _, n := range s.NodeCache {
		if n == nil {
			continue
		}
		chance := chanceFn()
		if rand.Float64() >= chance {
			continue
		}
		idx := n.nextFreeDataIndex()
		if idx < 0 {
			continue
		}
		s.createData(createDataFn, idx)
		n.applyNewData(idx)
	}
}

func (s *Simulation) createNode() *Node {
	nodeInitDataFn := s.NodeInitialDataFactoryFn()
	allocateNDataToNodeFn := s.AllocateNDataToNodeFactoryFn()
	createDataFn := s.CreateDataFactoryFn()
	nodeMaxBSizeFn := s.NodeMaxBSizeFactoryFn()
	waitActivityFn := s.WaitActivityFactoryFn()
	peerListFn := s.PeerListFactoryFn()

	// not concurrent safe
	dcSize := allocateNDataToNodeFn(s.NDataCacheFree)
	s.NDataCacheFree -= dcSize
	indices := make([]int, 0, dcSize)
	for idx, allocd := range s.DataAllocdToNode {
		if !allocd {
			indices = append(indices, idx)
			s.DataAllocdToNode[idx] = true
		}
		if len(indices) >= dcSize {
			break
		}
	}
	size := 0
	nInitData := nodeInitDataFn(dcSize)
	for j := 0; j < nInitData && j < len(indices); j++ {
		size += s.createData(createDataFn, indices[j]).DataSize
	}
	// TODO: Log
	return NewNode(
		s.DataCache,
		indices,
		nodeMaxBSizeFn(size),
		waitActivityFn(),
		peerListFn())
}

func (s *Simulation) removeNode(r *Node) {
	idxR := -1
	for idx, n := range s.NodeCache {
		if n == r {
			idxR = idx
			break
		}
	}
	for _, dataIdx := range s.NodeCache[idxR].DataIndices {
		s.removeData(dataIdx)
		s.DataAllocdToNode[dataIdx] = false
	}
	s.NodeCache[idxR] = nil
}

func (s *Simulation) createData(createDataFn CreateDataFn, idx int) *Data {
	s.DataCache[idx] = NewData(createDataFn())
	return s.DataCache[idx]
}

func (s *Simulation) removeData(idx int) {
	s.DataCache[idx] = nil
}

func (s *Simulation) RLock() {
	s.mu.RLock()
}

func (s *Simulation) RUnlock() {
	s.mu.RUnlock()
}

func (s *Simulation) Quit() {
	s.doneCh <- true
	<-s.ackDoneCh
}

func (s *Simulation) Play() {
	s.playCh <- true
}

func (s *Simulation) Pause() {
	s.pauseCh <- true
}

func (s *Simulation) Run() {
	if !s.vizOnly {
		var err error
		s.Log, err = os.OpenFile("log.txt", os.O_RDWR|os.O_CREATE, 0755)
		if err != nil {
			panic(err)
		}
		s.NodeFile, err = os.OpenFile("node.txt", os.O_RDWR|os.O_CREATE, 0755)
		if err != nil {
			panic(err)
		}
		s.NodeStateFile, err = os.OpenFile("states.txt", os.O_RDWR|os.O_CREATE, 0755)
		if err != nil {
			panic(err)
		}
		fmt.Fprintf(s.NodeStateFile, "%s,%s,%s,%s,%s\n", "iter", "join", "wait", "xData", "askPeer")
		s.FxFile, err = os.OpenFile("fx.txt", os.O_RDWR|os.O_CREATE, 0755)
		if err != nil {
			panic(err)
		}
		fmt.Fprintf(s.FxFile, "%s,%s,%s,%s,%s,%s\n", "iter", "fx", "fx^2", "n", "avg", "stddev")
	}
	go func() {
		defer func() { s.ackDoneCh <- true }()
		ticker := time.NewTicker(nTickMilli * time.Millisecond)
		defer ticker.Stop()
		if !s.vizOnly {
			defer s.Log.Close()
			defer s.NodeFile.Close()
			defer s.NodeStateFile.Close()
			defer s.FxFile.Close()
		}
		i := 0
		for {
			select {
			case <-s.doneCh:
				return
			case <-s.pauseCh:
				select {
				case <-s.doneCh:
					return
				case <-s.playCh:
				}
			case _ = <-ticker.C:
				start := time.Now()
				s.mu.Lock()
				startPostLock := time.Now()
				if !s.vizOnly && isHopHistIter(i) {
					s.computeHopHist(i)
				}
				s.tick(i)
				s.tock(i)
				fx, fxsq, nfx := s.computeFxStatistics()
				var avg float64
				var stddev float64
				if nfx > 0 {
					avg = fx / float64(nfx)
					stddev = fxsq/float64(nfx) - (avg * avg)
				}
				if !s.vizOnly {
					s.writeNodeStateFile(i)
					s.writeNodeFile(i)
					s.writeFxFile(i, fx, fxsq, nfx, avg, stddev)
				}
				s.mu.Unlock()
				f := time.Now()
				s.redraw(i, int(math.Round(fx)), nfx, avg, stddev, f.Sub(start), f.Sub(startPostLock))
				i++
			}
		}
	}()
}

// tick progresses node states.
func (s *Simulation) tick(i int) {
	for _, n := range s.NodeCache {
		if n == nil {
			continue
		}
		summary := n.ApplyState(s)
		if len(summary) > 0 && !s.vizOnly {
			fmt.Fprintf(s.Log, "%d: %s\n", i, summary)
		}
	}
	for _, n := range s.NodeCache {
		if n == nil {
			continue
		}
		n.AdvanceState()
	}
}

// tock applies events to the ecosystem: nodes coming online or offline.
func (s *Simulation) tock(i int) {
	for _, t := range s.Tockers {
		t.Tock(s, i)
	}
}

func (s *Simulation) writeNodeFile(i int) {
	n := s.NodeCache[0]
	locs := n.getDataLocations()
	fmt.Fprintf(s.NodeFile, "%v,%v,%d,%v\n", i, n.Location, len(locs), locs)
}

func (s *Simulation) writeNodeStateFile(i int) {
	m := []int{
		/*StateJoin=*/ 0,
		/*StateWait=*/ 0,
		/*StateExchangeData=*/ 0,
		/*StateAskPeer=*/ 0,
	}
	for _, n := range s.NodeCache {
		if n == nil {
			continue
		}
		n.CountLastState(m)
	}
	fmt.Fprintf(s.NodeStateFile, "%d,%d,%d,%d,%d\n",
		i,
		m[StateJoin],
		m[StateWait],
		m[StateExchangeData],
		m[StateAskPeer])
}

func (s *Simulation) computeFxStatistics() (fx float64, fxsq float64, nfx int) {
	for _, n := range s.NodeCache {
		if n == nil {
			continue
		}
		fx += n.fx
		fxsq += n.fxsq
		nfx += n.nfx
	}
	return
}

func (s *Simulation) writeFxFile(i int, fx, fxsq float64, nfx int, avg, stddev float64) {
	fmt.Fprintf(s.FxFile, "%v,%v,%v,%v,%v,%v\n", i, fx, fxsq, nfx, avg, stddev)
}

func (s *Simulation) computeHopHist(i int) {
	m := make(map[*Data]map[*Node]int, len(s.DataCache))
	// Seed m with no-hop nodes
	for _, n := range s.NodeCache {
		if n == nil {
			continue
		}
		for _, di := range n.DataIndices {
			d := s.DataCache[di]
			if d == nil {
				continue
			}
			m[d] = map[*Node]int{
				n: 0,
			}
		}
	}
	// Build up m with h+1 nodes
	// This outer loop caps the maximum number of hops possible (through every
	// node)
	for i := 0; i < len(s.NodeCache); i++ {
		// Repeatedly build up hops
		stillHopping := false
		for _, d := range s.DataCache {
			if d == nil {
				continue
			}
			nodeMap := m[d]
			for _, n := range s.NodeCache {
				if n == nil {
					continue
				}
				n.peers.IterateOverPeersWith(func(p *Node) {
					if p == nil {
						return
					}
					if _, ok := nodeMap[n]; ok {
						return
					} else if peerHops, ok := nodeMap[p]; ok {
						nodeMap[n] = peerHops + 1
						m[d] = nodeMap
						stillHopping = true
					}
				})
			}
		}
		if !stillHopping {
			break
		}
	}
	// Build up disjoint counts
	disj := make(map[*Data]int, len(s.DataCache))
	for _, d := range s.DataCache {
		if d == nil {
			continue
		}
		nodeMap := m[d]
		for _, n := range s.NodeCache {
			if n == nil {
				continue
			}
			if _, ok := nodeMap[n]; !ok {
				if c, ok := disj[d]; ok {
					disj[d] = c + 1
				} else {
					disj[d] = 1
				}
			}
		}
	}
	// Compute histograph
	hopsHist := make([]int, len(s.NodeCache))
	disjHist := make([]int, len(s.NodeCache))
	for _, nm := range m {
		for _, hops := range nm {
			hopsHist[hops] += 1
		}
	}
	for _, nDisj := range disj {
		disjHist[nDisj] += 1
	}
	// Output to file
	histF, err := os.OpenFile(fmt.Sprintf("hist_%d.txt", i), os.O_RDWR|os.O_CREATE, 0755)
	if err != nil {
		panic(err)
	}
	defer histF.Close()
	fmt.Fprintf(histF, "%s,%s\n", "hops", "count")
	for i, v := range hopsHist {
		fmt.Fprintf(histF, "%d,%d\n", i, v)
	}
	disjF, err := os.OpenFile(fmt.Sprintf("disj_%d.txt", i), os.O_RDWR|os.O_CREATE, 0755)
	if err != nil {
		panic(err)
	}
	defer disjF.Close()
	fmt.Fprintf(disjF, "%s,%s\n", "disjoint", "count")
	for i, v := range disjHist {
		fmt.Fprintf(disjF, "%d,%d\n", i, v)
	}
}

var _ coordinator = &Simulation{}

func (s *Simulation) FindOtherArbitraryNode(notMe *Node) (n *Node) {
	for n == nil || n == notMe {
		offset := rand.Intn(len(s.NodeCache))
		n = s.NodeCache[offset]
	}
	return
}
