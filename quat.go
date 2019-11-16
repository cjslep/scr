package scr

import (
	"fmt"
	"math"
	"math/rand"
)

// Q Is a quateRnIon
type Q struct {
	I float64
	J float64
	K float64
	R float64 // Real
}

func (q Q) Neg() Q {
	return Q{
		I: -q.I,
		J: -q.J,
		K: -q.K,
		R: -q.R,
	}
}

func (q Q) Add(o Q) Q {
	return Q{
		I: q.I + o.I,
		J: q.J + o.J,
		K: q.K + o.K,
		R: q.R + o.R,
	}
}

func (q Q) Sub(o Q) Q {
	return q.Add(o.Neg())
}

func (q Q) Mul(o Q) Q {
	return Q{
		R: q.R*o.R - q.I*o.I - q.J*o.J - q.K*o.K,
		I: q.R*o.I + q.I*o.R + q.J*o.K - q.K*o.J,
		J: q.R*o.J - q.I*o.K + q.J*o.R + q.K*o.I,
		K: q.R*o.K + q.I*o.J - q.J*o.I + q.K*o.R,
	}
}

func (q Q) MulScalar(v float64) Q {
	return q.Mul(Q{R: v})
}

func (q Q) Div(o Q) Q {
	d := o.R*o.R + o.I*o.I + o.J*o.J + o.K*o.K
	t0 := (q.R*o.R + q.I*o.I + q.J*o.J + q.K*o.K) / d
	t1 := (q.I*o.R + q.R*o.I + q.K*o.J + q.J*o.K) / d
	t2 := (q.J*o.R + q.K*o.I + q.R*o.J + q.I*o.K) / d
	t3 := (q.K*o.R + q.J*o.I + q.I*o.J + q.R*o.K) / d
	return Q{
		I: t1,
		J: t2,
		K: t3,
		R: t0,
	}
}

func (q Q) DivScalar(v float64) Q {
	return q.Div(Q{R: v})
}

func (q Q) Conj() Q {
	return Q{
		I: -q.I,
		J: -q.J,
		K: -q.K,
		R: q.R,
	}
}

func (q Q) Norm() float64 {
	return math.Sqrt(q.R*q.R +
		q.I*q.I +
		q.J*q.J +
		q.K*q.K)
}

func (q Q) Dist(o Q) float64 {
	return q.Sub(o).Norm()
}

func (q Q) Unit() Q {
	return q.DivScalar(q.Norm())
}

func (q Q) Rotate(v V) V {
	p := Q{
		R: 0,
		I: v.X,
		J: v.Y,
		K: v.Z,
	}
	R := q.Mul(p).Mul(q.Conj())
	return V{
		X: R.I,
		Y: R.J,
		Z: R.K,
	}
}

func (q Q) String() string {
	return fmt.Sprintf("{%5.2f, %5.2f, %5.2f, %5.2f}", q.R, q.I, q.J, q.K)
}

// TODO: Needed?
func RandomQuaternion() Q {
	s := rand.Float64()
	sig1 := math.Sqrt(1 - s)
	sig2 := math.Sqrt(s)
	t1 := 2 * math.Pi * rand.Float64()
	t2 := 2 * math.Pi * rand.Float64()
	return Q{
		R: math.Cos(t2) * sig2,
		I: math.Sin(t1) * sig1,
		J: math.Cos(t1) * sig1,
		K: math.Sin(t2) * sig2,
	}
}
