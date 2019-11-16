package scr

import (
	"testing"
)

func TestMonteCarlo(t *testing.T) {
	existing := []V{
		V{0, 0, -1},
		V{0, -1, 0},
		V{-1, 0, 0},
		V{0, 1, 0},
		V{0, 0, 1},
		V{1, 0, 0},
	}
	weights := []float64{1, 1, 1, 1, 1, 1}
	v := MonteCarloMinimizer(existing, weights)
	t.Log(v)
}
