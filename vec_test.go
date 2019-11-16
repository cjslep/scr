package scr

import (
	"math"
	"testing"
)

func TestGreatCircleDistance90(t *testing.T) {
	v := V{0, 0, 1}
	o := V{0, 1, 0}
	if r := v.GreatCircleDistance(o); r != math.Pi/2 {
		t.Fatalf("%v != %v", r, math.Pi/2)
	}
}

func TestGreatCircleDistance180(t *testing.T) {
	v := V{0, 0, 1}
	o := V{0, 0, -1}
	if r := v.GreatCircleDistance(o.Unit()); r != math.Pi {
		t.Fatalf("%v != %v", r, math.Pi)
	}
}
