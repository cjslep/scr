package scr

import (
	"encoding/binary"
	"fmt"
	"math"
)

// V is a vector
type V struct {
	X float64
	Y float64
	Z float64
}

func (v V) Add(o V) V {
	return V{
		X: v.X + o.X,
		Y: v.Y + o.Y,
		Z: v.Z + o.Z,
	}
}

func (v V) Sub(o V) V {
	return V{
		X: v.X - o.X,
		Y: v.Y - o.Y,
		Z: v.Z - o.Z,
	}
}

func (v V) Dot(o V) float64 {
	return v.X*o.X + v.Y*o.Y + v.Z*o.Z
}

func (v V) Cross(o V) V {
	return V{
		X: v.Y*o.Z - v.Z*o.Y,
		Y: v.Z*o.X - v.X*o.Z,
		Z: v.X*o.Y - v.Y*o.X,
	}
}

func (v V) MulScalar(o float64) V {
	return V{
		X: v.X * o,
		Y: v.Y * o,
		Z: v.Z * o,
	}
}

func (v V) DivScalar(o float64) V {
	return V{
		X: v.X / o,
		Y: v.Y / o,
		Z: v.Z / o,
	}
}

// Norm is the 2 norm or the Euclidean norm
func (v V) Norm() float64 {
	return math.Sqrt(v.X*v.X + v.Y*v.Y + v.Z*v.Z)
}

func (v V) GreatCircleDistance(o V) float64 {
	d := v.Dot(o)
	if d == 0 {
		return math.Pi / 2
	}
	dist := math.Atan(v.Cross(o).Norm() / d)
	if d < 0 {
		return math.Pi + dist
	}
	return dist
}

func (v V) Unit() V {
	return v.DivScalar(v.Norm())
}

func (v V) Rotate(q Q) V {
	return q.Rotate(v)
}

func (v V) String() string {
	return fmt.Sprintf("{%v, %v, %v}", v.X, v.Y, v.Z)
}

func (v V) RawBytes() []byte {
	var buf [24]byte
	binary.BigEndian.PutUint64(buf[:8], math.Float64bits(v.X))
	binary.BigEndian.PutUint64(buf[8:16], math.Float64bits(v.Y))
	binary.BigEndian.PutUint64(buf[16:], math.Float64bits(v.Z))
	return buf[:]
}

func (v V) Project() (x, y float64) {
	x = v.X / (1 - v.Z)
	y = v.Y / (1 - v.Z)
	return
}

func (v V) ProjectGSD() (x, y float64) {
	if v.X == 0 && v.Y == 0 {
		if v.Z == 1 {
			x = 0
			y = 0
			return
		} else if v.Z == -1 {
			x = 0
			y = math.Pi
			return
		}
	}
	x, y = v.Project()
	n := V{X: x, Y: y}
	scaled := n.Unit().MulScalar(v.GreatCircleDistance(V{Z: 1}))
	return scaled.X, scaled.Y
}

func (v V) ShortestRotation(o V) Q {
	d := v.Dot(o)
	if d > 0.999999 {
		return Q{I: math.Pi, J: 0, K: 0, R: 0}
	} else if d < -0.999999 {
		return Q{I: 0, J: 0, K: 0, R: 1}
	} else {
		v0 := v.Cross(o)
		return Q{I: v0.X, J: v0.Y, K: v0.Z, R: d}
	}
}

func (v V) Equals(o V) bool {
	return v.X == o.X && v.Y == o.Y && v.Z == o.Z
}

// Random unit vector on a sphere
// TODO: needed?
func RandomVector() V {
	v := V{X: 1}
	return v.Rotate(RandomQuaternion())
}
