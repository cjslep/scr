package main

import (
	"flag"
	"fmt"
	"math"
	"math/rand"
	"time"

	"github.com/andlabs/ui"
	"github.com/cjslep/scr"
)

const (
	nMaxPeerSpreadDefault = 16
	nThenAfterClosest     = 8
	relaxedIter           = 5000
)

var nInitialNodes = flag.Int("n_init_nodes", 100, "Initial number of nodes to simulate")
var nMaxData = flag.Int("n_max_data", 10000000, "Maximum number of pieces of data to simulate")
var nMaxNodes = flag.Int("n_max_nodes", 1000, "Maximum number of nodes to simulate")
var vizOnly = flag.Bool("viz", false, "Only displays the UI when enabled")

var expNodeJoin = flag.Bool("exp_node_join", false, fmt.Sprintf("Run experiment with a node joining at iteration %d", relaxedIter))
var expNodeLeave = flag.Bool("exp_node_leave", false, fmt.Sprintf("Run experiment with a node leaving at iteration %d", relaxedIter))
var expGenerateDataAfterRelax = flag.Bool("exp_gen_data_after_relax", false, fmt.Sprintf("Run experiment with nodes generating new data after iteration %d @ 1%%", relaxedIter))
var expGenerateDataAfterRelax2 = flag.Bool("exp_gen_data_after_relax_2", false, fmt.Sprintf("Run experiment with nodes generating new data after iteration %d @ 2%%", relaxedIter))
var expProd = flag.Bool("exp_prod", false, fmt.Sprintf("Run experiment with nodes joining 5% of the time and data growth beginning at iteration %d, nodes can leave beginning 2500 iterations later", relaxedIter))

var peerClosest = flag.Bool("peer_closest", false, "Enable closest-peer network")
var peerMaxThenClosest = flag.Bool("peer_max_spread_then_closest", false, "Enable closest-peer after max-spread-peer network")

func main() {
	flag.Parse()
	s := CheckFlags()
	if err := ui.Main(setup(s)); err != nil {
		panic(err)
	}
}

func CheckFlags() (s *scr.Simulation) {
	n := 0
	var t []scr.Tocker
	dataGrowth := uncertainNormalDistFloatFactoryFn(
		/*Std Dev's Mean & Std Dev*/
		0.003, 0.001,
		/*Mean's Mean & Std Dev*/
		0.01, 0.001)
	if *expNodeJoin {
		t = []scr.Tocker{
			&join15k{},
		}
		n++
	}
	if *expNodeLeave {
		t = []scr.Tocker{
			&leave15k{},
		}
		n++
	}
	if *expGenerateDataAfterRelax {
		t = []scr.Tocker{
			&generateDataAfterRelax{},
		}
		n++
	}
	if *expGenerateDataAfterRelax2 {
		t = []scr.Tocker{
			&generateDataAfterRelax{},
		}
		dataGrowth = uncertainNormalDistFloatFactoryFn(
			/*Std Dev's Mean & Std Dev*/
			0.003, 0.001,
			/*Mean's Mean & Std Dev*/
			0.02, 0.001)
		n++
	}
	if *expProd {
		t = []scr.Tocker{
			&prod{},
		}
		dataGrowth = uncertainNormalDistFloatFactoryFn(
			/*Std Dev's Mean & Std Dev*/
			0.003, 0.001,
			/*Mean's Mean & Std Dev*/
			0.02, 0.001)
		n++
	}

	np := 0
	peerListFactoryFn := func() func() scr.PeerList {
		return func() scr.PeerList {
			return scr.NewMaximizePeerSpread(nMaxPeerSpreadDefault)
		}
	}
	if *peerClosest {
		peerListFactoryFn = func() func() scr.PeerList {
			return func() scr.PeerList {
				return scr.NewClosestNeighbors(nMaxPeerSpreadDefault)
			}
		}
		np++
	}
	if *peerMaxThenClosest {
		peerListFactoryFn = func() func() scr.PeerList {
			return func() scr.PeerList {
				return scr.NewMaxSpreadThenClosestNeighbors(nMaxPeerSpreadDefault-nThenAfterClosest, nThenAfterClosest)
			}
		}
		np++
	}

	if n > 1 {
		panic("too many exp_* flags chosen")
	} else if np > 1 {
		panic("too many peer_* flags chosen")
	}
	s = scr.NewSimulation(*nInitialNodes,
		*nMaxData,
		*nMaxNodes,
		t,
		/* # of initial pieces of Data for a Node */
		cappedUncertainNormalDistFactoryFn(
			/*Std Dev's Mean & Std Dev*/
			2, 2,
			/*Mean's Mean & Std Dev*/
			100, 2,
		),
		/* # of maximum Data pieces of Data a Node can have */
		cappedUncertainNormalDistFactoryFn(
			/*Std Dev's Mean & Std Dev*/
			10, 10,
			/*Mean's Mean & Std Dev*/
			2000, 10,
		),
		/* # of bytes in a piece of Data (Datashards is always 32kb) */
		uncertainNormalDistByteFactoryFn(
			/*Std Dev's Mean & Std Dev*/
			0, 0,
			/*Mean's Mean & Std Dev*/
			32000, 0,
		),
		/* # of maximum bytes a Node can have */
		addedUncertainNormalDistFactoryFn(
			/*Std Dev's Mean & Std Dev*/
			1000000, 1000000,
			/*Mean's Mean & Std Dev*/
			1000000000, 1000000,
		),
		/* % of time the node waits (does nothing) */
		uncertainNormalDistFloatFactoryFn(
			/*Std Dev's Mean & Std Dev*/
			0.03, 0.01,
			/*Mean's Mean & Std Dev*/
			0.5, 0.01,
		),
		/* % chance nodes will grow data (if experiment enabled) */
		dataGrowth,
		/* Peer list factory function */
		peerListFactoryFn,
		*vizOnly)
	return
}

func setup(s *scr.Simulation) func() {
	return func() {
		mainwin := ui.NewWindow("scr demo", 640, 720, true)
		mainwin.SetMargined(false)
		mainwin.OnClosing(func(*ui.Window) bool {
			s.Quit()
			mainwin.Destroy()
			ui.Quit()
			return false
		})
		ui.OnShouldQuit(func() bool {
			mainwin.Destroy()
			return true
		})

		// Labels & visualizations
		iterLabel := ui.NewLabel("i=0")
		fxLabel := ui.NewLabel("fx=?")
		nfxLabel := ui.NewLabel("n=?")
		avgLabel := ui.NewLabel("μ=?")
		stddevLabel := ui.NewLabel("σ=?")
		tickLabel := ui.NewLabel("sim tick:")
		tickLabelDur := ui.NewLabel("?")
		tickLabelDurLockless := ui.NewLabel("?")
		vizLabel := ui.NewLabel("viz draw:")
		vizLabelDur := ui.NewLabel("?")
		vizLabelDurLockless := ui.NewLabel("?")
		visual := &viz{
			s:                   s,
			vizMode:             vizModeTwoNodes,
			vizLabelDur:         vizLabelDur,
			vizLabelDurLockless: vizLabelDurLockless,
			enableLink:          true,
			enableNode:          true,
			enableData:          true,
			enableInnerCircle:   true,
			enableOuterCircle:   true,
		}
		projection := ui.NewArea(visual)
		s.SetRedraw(func(i, fx, nfx int, avg, stddev float64, dur, durLockless time.Duration) {
			ui.QueueMain(func() {
				iterLabel.SetText(fmt.Sprintf("i=%d", i))
				fxLabel.SetText(fmt.Sprintf("fx=%d", fx))
				nfxLabel.SetText(fmt.Sprintf("n=%d", nfx))
				avgLabel.SetText(fmt.Sprintf("μ=%5.2f", avg))
				stddevLabel.SetText(fmt.Sprintf("σ=%5.2f", stddev))
				tickLabelDur.SetText(fmt.Sprintf("%s", dur))
				tickLabelDurLockless.SetText(fmt.Sprintf("%s", durLockless))
				projection.QueueRedrawAll()
			})
		})

		vbox := ui.NewVerticalBox()
		vbox.Append(projection, false)

		grid := ui.NewGrid()
		grid.SetPadded(true)
		grid.Append(iterLabel, 0, 0, 1, 1, false, ui.AlignFill, false, ui.AlignFill)
		grid.Append(fxLabel, 1, 0, 1, 1, false, ui.AlignFill, false, ui.AlignFill)
		grid.Append(nfxLabel, 2, 0, 1, 1, false, ui.AlignFill, false, ui.AlignFill)
		grid.Append(avgLabel, 3, 0, 1, 1, false, ui.AlignFill, false, ui.AlignFill)
		grid.Append(stddevLabel, 4, 0, 1, 1, false, ui.AlignFill, false, ui.AlignFill)

		grid.Append(ui.NewLabel("timed item"), 0, 1, 1, 1, false, ui.AlignFill, false, ui.AlignFill)
		grid.Append(ui.NewLabel("total duration"), 1, 1, 1, 1, false, ui.AlignFill, false, ui.AlignFill)
		grid.Append(ui.NewLabel("lockless duration"), 2, 1, 1, 1, false, ui.AlignFill, false, ui.AlignFill)

		grid.Append(tickLabel, 0, 2, 1, 1, false, ui.AlignFill, false, ui.AlignFill)
		grid.Append(tickLabelDur, 1, 2, 1, 1, false, ui.AlignFill, false, ui.AlignFill)
		grid.Append(tickLabelDurLockless, 2, 2, 1, 1, false, ui.AlignFill, false, ui.AlignFill)
		grid.Append(vizLabel, 0, 3, 1, 1, false, ui.AlignFill, false, ui.AlignFill)
		grid.Append(vizLabelDur, 1, 3, 1, 1, false, ui.AlignFill, false, ui.AlignFill)
		grid.Append(vizLabelDurLockless, 2, 3, 1, 1, false, ui.AlignFill, false, ui.AlignFill)
		vbox.Append(grid, false)

		// Viz Mode Buttons
		vizModeLabel := ui.NewLabel(vizModeTwoNodes)
		buttonVM1 := vizModeButton(vizModeTwoNodes, visual, vizModeLabel, projection)
		buttonVM2 := vizModeButton(vizModeNodes, visual, vizModeLabel, projection)
		buttonVM3 := vizModeButton(vizModeData, visual, vizModeLabel, projection)
		buttonVM4 := vizModeButton(vizModeDebugPoints, visual, vizModeLabel, projection)
		btnGrid := ui.NewGrid()
		btnGrid.Append(vizModeLabel, 0, 0, 1, 1, false, ui.AlignFill, false, ui.AlignFill)
		btnGrid.Append(buttonVM1, 0, 1, 1, 1, false, ui.AlignFill, false, ui.AlignFill)
		btnGrid.Append(buttonVM2, 1, 1, 1, 1, false, ui.AlignFill, false, ui.AlignFill)
		btnGrid.Append(buttonVM3, 2, 1, 1, 1, false, ui.AlignFill, false, ui.AlignFill)
		btnGrid.Append(buttonVM4, 3, 1, 1, 1, false, ui.AlignFill, false, ui.AlignFill)
		// Play Pause Buttons
		buttonPlayPause := ui.NewButton("Pause")
		buttonPlayPause.OnClicked(func(b *ui.Button) {
			if b.Text() == "Pause" {
				s.Pause()
				b.SetText("Play")
			} else {
				s.Play()
				b.SetText("Pause")
			}
		})
		btnGrid.Append(buttonPlayPause, 0, 2, 1, 1, false, ui.AlignFill, false, ui.AlignFill)
		vbox.Append(btnGrid, false)

		mainwin.SetChild(vbox)
		mainwin.Show()
		s.Run()
	}
}

// cappedUncertainNormalDistFactoryFn uses uncertainty in the mean and
// distribution itself to generate slightly different probability distributions
// when called for.
//
// Results in more accurate simulations that reward reducing uncertainty.
func cappedUncertainNormalDistFactoryFn(
	// Uncertainty in the standard deviation
	uncertaintyStdDevMean,
	uncertaintyStdDevStdDev,
	// Uncertainty in the mean
	uncertaintyMeanMean,
	uncertaintyMeanStdDev float64) func() func(int) int {
	return func() func(int) int {
		uncertainStdDev := rand.NormFloat64()*uncertaintyStdDevStdDev + uncertaintyStdDevMean
		uncertainMean := rand.NormFloat64()*uncertaintyMeanStdDev + uncertaintyMeanMean
		return func(max int) int {
			sample := rand.NormFloat64()*uncertainStdDev + uncertainMean
			return int(math.Min(math.Floor(sample), float64(max)))
		}
	}
}

func addedUncertainNormalDistFactoryFn(
	// Uncertainty in the standard deviation
	uncertaintyStdDevMean,
	uncertaintyStdDevStdDev,
	// Uncertainty in the mean
	uncertaintyMeanMean,
	uncertaintyMeanStdDev float64) func() func(int) int {
	return func() func(int) int {
		uncertainStdDev := rand.NormFloat64()*uncertaintyStdDevStdDev + uncertaintyStdDevMean
		uncertainMean := rand.NormFloat64()*uncertaintyMeanStdDev + uncertaintyMeanMean
		return func(curr int) int {
			sample := rand.NormFloat64()*uncertainStdDev + uncertainMean
			return int(math.Floor(sample + float64(curr)))
		}
	}
}

func uncertainNormalDistByteFactoryFn(
	// Uncertainty in the standard deviation
	uncertaintyStdDevMean,
	uncertaintyStdDevStdDev,
	// Uncertainty in the mean
	uncertaintyMeanMean,
	uncertaintyMeanStdDev float64) func() scr.CreateDataFn {
	return func() scr.CreateDataFn {
		uncertainStdDev := rand.NormFloat64()*uncertaintyStdDevStdDev + uncertaintyStdDevMean
		uncertainMean := rand.NormFloat64()*uncertaintyMeanStdDev + uncertaintyMeanMean
		return func() []byte {
			sample := rand.NormFloat64()*uncertainStdDev + uncertainMean
			l := int(math.Max(math.Floor(sample), 1))
			b := make([]byte, l)
			_, _ = rand.Read(b)
			return b
		}
	}
}

func uncertainNormalDistFloatFactoryFn(
	// Uncertainty in the standard deviation
	uncertaintyStdDevMean,
	uncertaintyStdDevStdDev,
	// Uncertainty in the mean
	uncertaintyMeanMean,
	uncertaintyMeanStdDev float64) func() func() float64 {
	return func() func() float64 {
		uncertainStdDev := rand.NormFloat64()*uncertaintyStdDevStdDev + uncertaintyStdDevMean
		uncertainMean := rand.NormFloat64()*uncertaintyMeanStdDev + uncertaintyMeanMean
		return func() float64 {
			sample := rand.NormFloat64()*uncertainStdDev + uncertainMean
			return math.Max(math.Min(sample, 1.0), 0)
		}
	}
}

// UI Helpers

func vizModeButton(vm string, v *viz, l *ui.Label, proj *ui.Area) *ui.Button {
	button := ui.NewButton(vm)
	button.OnClicked(func(*ui.Button) {
		v.vizMode = vm
		l.SetText(vm)
		proj.QueueRedrawAll()
	})
	return button
}

// scr.Tockers

var _ scr.Tocker = &join15k{}

type join15k struct{}

func (*join15k) Tock(s *scr.Simulation, i int) {
	if i == relaxedIter {
		s.NewNodeJoins()
	}
}

var _ scr.Tocker = &leave15k{}

type leave15k struct{}

func (*leave15k) Tock(s *scr.Simulation, i int) {
	if i == relaxedIter {
		s.ExistingNodeLeaves()
	}
}

var _ scr.Tocker = &generateDataAfterRelax{}

type generateDataAfterRelax struct{}

func (*generateDataAfterRelax) Tock(s *scr.Simulation, i int) {
	if i > relaxedIter {
		s.GenerateLocalData()
	}
}

var _ scr.Tocker = &prod{}

type prod struct{}

func (*prod) Tock(s *scr.Simulation, i int) {
	if i > relaxedIter {
		s.GenerateLocalData()
		ch := rand.Intn(100)
		if ch < 2 {
			r := rand.Intn(2)
			if r == 0 && i > relaxedIter+1000 {
				s.ExistingNodeLeaves()
			} else {
				s.NewNodeJoins()
			}
		}
	}
}
