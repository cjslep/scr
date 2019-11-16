package main

import (
	"fmt"
	"math"
	"time"

	"github.com/andlabs/ui"
	"github.com/cjslep/scr"
)

const (
	scaleFactor  = 6.7
	colorWhite   = 0xFFFFFF
	colorBlack   = 0x000000
	colorRed     = 0xFF0000
	colorYellow  = 0xFFFF00
	colorGreen   = 0x00FF00
	colorBlue    = 0x0000FF
	colorGrey    = 0x888888
	colorMagenta = 0xFF00FF
	nNodes       = 100
	nData        = 2
)

var _ ui.AreaHandler = &viz{}

func NewSolidBrush(color uint32, alpha float64) *ui.DrawBrush {
	brush := &ui.DrawBrush{}
	brush.Type = ui.DrawBrushTypeSolid
	component := uint8((color >> 16) & 0xFF)
	brush.R = float64(component) / 255
	component = uint8((color >> 8) & 0xFF)
	brush.G = float64(component) / 255
	component = uint8(color & 0xFF)
	brush.B = float64(component) / 255
	brush.A = alpha
	return brush
}

const (
	vizModeTwoNodes    = "Highlight Two Nodes And Their Data"
	vizModeNodes       = "Draw All Nodes"
	vizModeData        = "Draw All Data"
	vizModeDebugPoints = "Draw Debug Points"
)

type viz struct {
	s                   *scr.Simulation
	vizMode             string
	vizLabelDur         *ui.Label
	vizLabelDurLockless *ui.Label
	enableLink          bool
	enableNode          bool
	enableData          bool
	enableInnerCircle   bool
	enableOuterCircle   bool
}

func (v *viz) Draw(a *ui.Area, p *ui.AreaDrawParams) {
	start := time.Now()
	v.s.RLock()
	defer v.s.RUnlock()
	startPostLock := time.Now()
	// Draw Background
	bgBrush := NewSolidBrush(colorBlack, 1.0)
	path := ui.DrawNewPath(ui.DrawFillModeWinding)
	path.AddRectangle(0, 0, p.AreaWidth, p.AreaHeight)
	path.End()
	p.Context.Fill(path, bgBrush)
	path.Free()

	sp := &ui.DrawStrokeParams{
		Cap:        ui.DrawLineCapFlat,
		Join:       ui.DrawLineJoinMiter,
		Thickness:  2,
		MiterLimit: ui.DrawDefaultMiterLimit,
	}
	spThin := &ui.DrawStrokeParams{
		Cap:        ui.DrawLineCapFlat,
		Join:       ui.DrawLineJoinMiter,
		Thickness:  1,
		MiterLimit: ui.DrawDefaultMiterLimit,
	}

	iterScale := p.AreaWidth / scaleFactor
	if p.AreaHeight < p.AreaWidth {
		iterScale = p.AreaHeight / scaleFactor
	}
	adjustXFn := func(x float64) float64 {
		xi := x + p.AreaWidth/2
		xi += x * iterScale
		return math.Round(xi)
	}
	adjustYFn := func(y float64) float64 {
		yi := p.AreaHeight/2 - y
		yi -= y * iterScale
		return math.Round(yi)
	}

	// Draw inner and outer circles
	if v.enableInnerCircle {
		circleBrush := NewSolidBrush(colorGrey, 1.0)
		path = ui.DrawNewPath(ui.DrawFillModeWinding)
		x := adjustXFn(math.Pi / 2)
		y := adjustYFn(0)
		x0 := adjustXFn(0)
		y0 := y
		path.NewFigure(x, y)
		path.ArcTo(x0, y0, (math.Pi/2)*(iterScale), 0, math.Pi*2, false)
		path.End()
		p.Context.Stroke(path, circleBrush, spThin)
		path.Free()
	}
	if v.enableOuterCircle {
		circleBrush := NewSolidBrush(colorGrey, 1.0)
		path = ui.DrawNewPath(ui.DrawFillModeWinding)
		x := adjustXFn(math.Pi)
		y := adjustYFn(0)
		x0 := adjustXFn(0)
		y0 := y
		path.NewFigure(x, y)
		path.ArcTo(x0, y0, math.Pi*(iterScale), 0, math.Pi*2, false)
		path.End()
		p.Context.Stroke(path, circleBrush, spThin)
		path.Free()
	}

	if v.vizMode == vizModeTwoNodes {
		// Draw Nodes-Data Lines
		nodesToDraw := make([]*scr.Node, 0, nNodes)
		dataToDraw := make([]*scr.Data, 0, nData*100*nNodes)
		dataDiv := 0
		for i, n := range v.s.NodeCache {
			if n == nil {
				continue
			}
			if len(nodesToDraw) >= nNodes {
				break
			} else {
				nodesToDraw = append(nodesToDraw, n)
			}
			if i < nData {
				dataDiv = len(dataToDraw)
				for _, di := range n.DataIndices {
					if v.s.DataCache[di] == nil {
						continue
					}
					dataToDraw = append(dataToDraw, v.s.DataCache[di])
				}
			}
		}

		if v.enableLink {
			lineBrush := NewSolidBrush(colorBlue, 1.0)
			path = ui.DrawNewPath(ui.DrawFillModeWinding)
			n := nodesToDraw[0]
			xProj, yProj := n.Location.ProjectGSD()
			for _, l := range n.PeerLocations() {
				xProj0, yProj0 := l.ProjectGSD()
				x := adjustXFn(xProj)
				y := adjustYFn(yProj)
				x0 := adjustXFn(xProj0)
				y0 := adjustYFn(yProj0)
				path.NewFigure(x, y)
				path.LineTo(x0, y0)
			}
			path.End()
			p.Context.Stroke(path, lineBrush, spThin)
			path.Free()
		}

		// Draw Nodes
		if v.enableNode {
			// Node 0
			path = ui.DrawNewPath(ui.DrawFillModeWinding)
			nodeBrush0 := NewSolidBrush(colorGreen, 1.0)
			xProj, yProj := nodesToDraw[0].Location.ProjectGSD()
			y0 := adjustYFn(yProj)
			x0 := adjustXFn(xProj)
			x := x0 + 3
			y := y0
			path.NewFigure(x, y)
			path.ArcTo(x0, y0, 3, 0, math.Pi*2, false)
			path.End()
			p.Context.Stroke(path, nodeBrush0, spThin)
			path.Free()
			// Node 1
			path = ui.DrawNewPath(ui.DrawFillModeWinding)
			nodeBrush1 := NewSolidBrush(colorYellow, 1.0)
			xProj, yProj = nodesToDraw[1].Location.ProjectGSD()
			y0 = adjustYFn(yProj)
			x0 = adjustXFn(xProj)
			x = x0 + 3
			y = y0
			path.NewFigure(x, y)
			path.ArcTo(x0, y0, 3, 0, math.Pi*2, false)
			path.End()
			p.Context.Stroke(path, nodeBrush1, spThin)
			path.Free()
			// Rest
			nodeBrush := NewSolidBrush(colorRed, 1.0)
			path = ui.DrawNewPath(ui.DrawFillModeWinding)
			for i, n := range nodesToDraw {
				if i > 1 {
					xProj, yProj = n.Location.ProjectGSD()
					x = adjustXFn(xProj)
					y = adjustYFn(yProj)
					path.NewFigure(x, y)
					path.LineTo(x, y+1)
				}
			}
			path.End()
			p.Context.Stroke(path, nodeBrush, sp)
			path.Free()
		}

		// Draw Data
		if v.enableData {
			// Data 0
			dataBrush := NewSolidBrush(colorGreen, 1.0)
			path = ui.DrawNewPath(ui.DrawFillModeWinding)
			for i, d := range dataToDraw {
				if i >= dataDiv {
					break
				}
				xProj, yProj := d.Location.ProjectGSD()
				x := adjustXFn(xProj)
				y := adjustYFn(yProj)
				path.NewFigure(x, y)
				path.LineTo(x, y+1)
			}
			path.End()
			p.Context.Stroke(path, dataBrush, sp)
			path.Free()
			// Data 1
			dataBrush = NewSolidBrush(colorYellow, 1.0)
			path = ui.DrawNewPath(ui.DrawFillModeWinding)
			for i, d := range dataToDraw {
				if i < dataDiv {
					continue
				}
				xProj, yProj := d.Location.ProjectGSD()
				x := adjustXFn(xProj)
				y := adjustYFn(yProj)
				path.NewFigure(x, y)
				path.LineTo(x, y+1)
			}
			path.End()
			p.Context.Stroke(path, dataBrush, sp)
			path.Free()

		}
	} else if v.vizMode == vizModeNodes {
		nodeBrush := NewSolidBrush(colorRed, 1.0)

		path = ui.DrawNewPath(ui.DrawFillModeWinding)
		for _, n := range v.s.NodeCache {
			if n == nil {
				continue
			}
			xProj, yProj := n.Location.ProjectGSD()
			x := adjustXFn(xProj)
			y := adjustYFn(yProj)
			path.NewFigure(x, y)
			path.LineTo(x, y+1)
		}
		path.End()
		p.Context.Stroke(path, nodeBrush, sp)
		path.Free()
	} else if v.vizMode == vizModeData {
		nodeBrush := NewSolidBrush(colorYellow, 1.0)

		path = ui.DrawNewPath(ui.DrawFillModeWinding)
		for _, d := range v.s.DataCache {
			if d == nil {
				continue
			}
			xProj, yProj := d.Location.ProjectGSD()
			x := adjustXFn(xProj)
			y := adjustYFn(yProj)
			path.NewFigure(x, y)
			path.LineTo(x, y+1)
		}
		path.End()
		p.Context.Stroke(path, nodeBrush, sp)
		path.Free()
	} else if v.vizMode == vizModeDebugPoints {
		spThick := &ui.DrawStrokeParams{
			Cap:        ui.DrawLineCapFlat,
			Join:       ui.DrawLineJoinMiter,
			Thickness:  3,
			MiterLimit: ui.DrawDefaultMiterLimit,
		}

		colors := []uint32{
			colorWhite,
			colorRed,
			colorYellow,
			colorGreen,
			colorBlue,
			colorMagenta,
		}
		locs := []scr.V{
			{1, 0, 0},
			{0, 1, 0},
			{0, 0, 1},
			{-1, 0, 0},
			{0, -1, 0},
			{0, 0, -1},
		}

		for i := 0; i < len(locs) && i < len(colors); i++ {
			xProj, yProj := locs[i].ProjectGSD()
			x := adjustXFn(xProj)
			y := adjustYFn(yProj)
			path := ui.DrawNewPath(ui.DrawFillModeWinding)
			path.NewFigure(x, y)
			path.LineTo(x, y+1)
			path.End()
			brush := NewSolidBrush(colors[i], 1.0)
			p.Context.Stroke(path, brush, spThick)
			path.Free()
		}

		const os3 = 0.577350269189625764509148780501957455647601751270126876018
		qlocs := []scr.V{
			{os3, os3, os3},
			{-os3, os3, -os3},
			{-os3, -os3, os3},
			{-os3, -os3, -os3},
			{os3, -os3, os3},
			{os3, os3, -os3},
		}
		for i := 0; i < len(qlocs) && i < len(colors); i++ {
			xProj, yProj := qlocs[i].ProjectGSD()
			x := adjustXFn(xProj)
			y := adjustYFn(yProj)
			path := ui.DrawNewPath(ui.DrawFillModeWinding)
			path.NewFigure(x, y)
			path.LineTo(x, y+1)
			path.End()
			brush := NewSolidBrush(colors[i], 1.0)
			p.Context.Stroke(path, brush, spThick)
			path.Free()
		}

	}

	f := time.Now()
	v.vizLabelDur.SetText(fmt.Sprintf("%s", f.Sub(start)))
	v.vizLabelDurLockless.SetText(fmt.Sprintf("%s", f.Sub(startPostLock)))
}

func (v *viz) MouseEvent(a *ui.Area, me *ui.AreaMouseEvent) {
	// Nothing
}

func (v *viz) MouseCrossed(a *ui.Area, left bool) {
	// Nothing
}

func (v *viz) DragBroken(a *ui.Area) {
	// Nothing
}

func (v *viz) KeyEvent(a *ui.Area, ke *ui.AreaKeyEvent) (handled bool) {
	// do not handle any keys
	return false
}
