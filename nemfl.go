package scr

import (
	"fmt"
)

// See documentation for solveNonEuclideanMultifacilityLocation.
func SolveNonEuclideanMultifacilityLocation(
	existingLocations []V,
	existingLocationWeights []float64,
	nonsmoothTolerance, smoothTolerance float64) (V, error) {
	result, x0, alpha0, _, _, _ := solveNonEuclideanMultifacilityLocationNonSmooth(
		existingLocations,
		existingLocationWeights,
		nonsmoothTolerance)
	if !result.Equals(noAnswer) {
		return result, nil
	}
	smoothSolution, _, _, _ := solveNonEuclideanMultifacilityLocationSmooth(
		existingLocations,
		existingLocationWeights,
		smoothTolerance,
		x0,
		alpha0)
	if smoothSolution == noAnswer {
		return V{}, fmt.Errorf("No NEMFL solution")
	}
	return smoothSolution, nil
}

// See documentation for solveNonEuclideanMultifacilityLocation.
func SolveNonEuclideanMultifacilityLocationSkipNonSmooth(
	existingLocations []V,
	existingLocationWeights []float64,
	nonsmoothTolerance, smoothTolerance float64,
	initialPoint V) (V, error) {
	smoothSolution, _, _, _ := solveNonEuclideanMultifacilityLocationSmooth(
		existingLocations,
		existingLocationWeights,
		smoothTolerance,
		initialPoint,
		0.001)
	if smoothSolution == noAnswer {
		return V{}, fmt.Errorf("No NEMFL solution")
	}
	return smoothSolution, nil
}

// See documentation for solveNonEuclideanMultifacilityLocation.
func SolveNonEuclideanMultifacilityLocationMonteCarlo(
	existingLocations []V,
	existingLocationWeights []float64,
	nonsmoothTolerance, smoothTolerance float64,
	nMC int) (V, float64, float64, int, error) {
	result, x0, alpha0, fxaj, fxsq, nfx := solveNonEuclideanMultifacilityLocationNonSmooth(
		existingLocations,
		existingLocationWeights,
		nonsmoothTolerance)
	if !result.Equals(noAnswer) {
		return result, fxaj, fxsq, nfx, nil
	}
	smoothSolution := noAnswer
	var fxk float64
	for i := 0; i < nMC; i++ {
		if i == 0 {
			si, fxki, fxksq, nfxk := solveNonEuclideanMultifacilityLocationSmooth(
				existingLocations,
				existingLocationWeights,
				smoothTolerance,
				x0,
				alpha0)
			smoothSolution = si
			fxk = fxki
			fxsq = fxksq
			nfx = nfxk
		} else {
			si, fxki, fxksq, nfxk := solveNonEuclideanMultifacilityLocationSmooth(
				existingLocations,
				existingLocationWeights,
				smoothTolerance,
				RandomVector(),
				0.001)
			if !si.Equals(noAnswer) && (smoothSolution.Equals(noAnswer) || fxki < fxk) {
				smoothSolution = si
				fxk = fxki
				fxsq = fxksq
				nfx = nfxk
			}
		}
	}
	if smoothSolution == noAnswer {
		return V{}, fxk, fxsq, nfx, fmt.Errorf("No NEMFL solution")
	}
	return smoothSolution, fxk, fxsq, nfx, nil
}

const (
	maxAlpha = 1000
	maxK     = 100000
)

var (
	noAnswer = V{2, 2, 2}
)

// solveNonEuclideanMultifacilityLocationNonSmooth implements the algorithm in
// the paper titled:
//
//    "A Globally Convergent Algorithm for Facility Location on a Sphere"
//    by G.-L. XUE
//    Computers Math. Applic. Vol. 27, No. 6, pp. 37-50, 1994
//
// Solves the non-smooth problem only.
//
// The non-smooth solutions are the ones that examine the existing
// locations themselves. In practice, it may be better to skip
// checking these locations. However, the optimization function is non-
// differentiable at these locations, so in the rare case collinearity
// along a great circle arc occurs, then the nonsmooth solution would be
// needed.
//
// Finds the optimal location related to other locations that minimizes a
// weighted great circle distance to each.
//
// Everything must lie on the unit sphere.
func solveNonEuclideanMultifacilityLocationNonSmooth(
	existingLocations []V, // Existing locations on a unit sphere
	existingLocationWeights []float64, // Must be positive
	// These tolerances can help bound the number of iterations while
	// maintaining a degree of accuracy. Must be nonnegative.
	nonsmoothTolerance float64) (result, x0 V, alpha0, fxaj, fxsq float64, nfx int) {
	// Step 1
	//
	// Check nonsmooth solutions, where smooth solutions are non-
	// differentiable.
	faj := make([]float64, len(existingLocations))
	fsqaj := make([]float64, len(existingLocations))
	for j := range existingLocations {
		faj[j], fsqaj[j] = geodesicDistancesFromI(j, existingLocations, existingLocationWeights)
	}
	var at0idx int
	for t, at := range existingLocations {
		isMin := true
		for j := range existingLocations {
			if j == t {
				continue
			}
			if faj[j] > faj[t] {
				isMin = false
				break
			}
		}
		if isMin {
			at0idx = t
			// Guaranteed to occur at least once. If occurs
			// more than once, we'll just get the last one
			// for the smooth search. That's OK, the
			// iterative process is convergent, so having an
			// arbitrary starting point is fine.
			if optimalityConditionFromI(t, existingLocations, existingLocationWeights, nonsmoothTolerance) {
				return at, V{}, 0, faj[t], fsqaj[t], len(existingLocations)
			}
		}
	}
	// Step 2
	//
	// Guess an initial point that is near the best nonsmooth
	// solution, from which we will begin the smooth search.
	//
	// We're given at0idx and at0 from Step 1 as a "minimum"
	foundX := false
	for !foundX {
		// First condition: that at0 plus a
		// rotation of alpha*dval is along the convex
		// hull of ajat.
		//
		// Second condition: A new points is at 'at' plus
		// alpha*dval.
		//
		// In practice, I could not get the paper's algoirthm
		// to work. Neither here in code, nor in pen/paper
		// aka spreadsheets would it work out.
		//
		// Random sampling for the win.
		xCandidate := RandomVector()
		fx, _ := geodesicDistances(xCandidate, existingLocations, existingLocationWeights)
		if fx < faj[at0idx] {
			x0 = xCandidate
			alpha0 = 0.001
			foundX = true
			break
		}
	}
	return noAnswer, x0, alpha0, 0.0, 0.0, 0
}

// solveNonEuclideanMultifacilityLocationSmooth implements the algorithm in the
// paper titled:
//
//    "A Globally Convergent Algorithm for Facility Location on a Sphere"
//    by G.-L. XUE
//    Computers Math. Applic. Vol. 27, No. 6, pp. 37-50, 1994
//
// Finds the optimal location related to other locations that minimizes a
// weighted great circle distance to each for smooth solutions.
//
// Everything must lie on the unit sphere.
func solveNonEuclideanMultifacilityLocationSmooth(
	existingLocations []V, // Existing locations on a unit sphere
	existingLocationWeights []float64, // Must be positive
	// These tolerances can help bound the number of iterations while
	// maintaining a degree of accuracy. Must be nonnegative.
	smoothTolerance float64,
	// 'x0' is an initial point on the unit sphere to begin searching for
	// the smooth solution. Only used if skipNonSmooth is true.
	x0 V,
	alpha0 float64) (V, float64, float64, int) {
	xk := x0
	alphak := alpha0
	// Should be convergent. But protect against unreasonable #s of iterations
	k := 1
	var prevFxk float64
	for ; k < maxK; k++ {
		// Step 3
		dk := dx(xk, existingLocations, existingLocationWeights)
		fxk, fxsqk := geodesicDistances(xk, existingLocations, existingLocationWeights)
		if k == 1 {
			prevFxk = fxk
		} else if prevFxk == fxk {
			// This occurs when a point is stuck in a local minima.
			return xk, fxk, fxsqk, len(existingLocations)
		} else {
			prevFxk = fxk
		}
		if optimalityCondition(xk, existingLocations, existingLocationWeights, smoothTolerance) {
			return xk, fxk, fxsqk, len(existingLocations)
		} else {
			alphak = alphax(xk, existingLocations, existingLocationWeights)
		}
		// Step 4
		//
		// This should be convergent. But protect against unreasonable #s of iterations.
		alphaIter := 1
		prevFxn := 0.0
		for ; alphaIter < maxAlpha; alphaIter++ {
			alphakdk := dk.MulScalar(alphak)
			xn := xk.Add(alphakdk).DivScalar(xk.Add(alphakdk).Norm())
			fxn, _ := geodesicDistances(xn, existingLocations, existingLocationWeights)
			if fxn <= fxk-0.1*alphak*dk.Norm()*dk.Norm() {
				xk = xn
				break
			} else {
				if alphaIter == 1 {
					prevFxn = fxn
				} else if prevFxn == fxn {
					// This occurs when a point is stuck at a local minima.
					return xk, fxk, fxsqk, len(existingLocations)
				}
				alphak *= 0.5
			}
		}
	}
	return noAnswer, 0, 0, 0
}

// geodesicDistances measures the sum of all great circle distances from an
// arbitrary point to a set of eval points. Everything must lie on the unit
// sphere.
//
// c is a parallel array of eval, for weights.
//
// Equation 1
func geodesicDistances(p V, eval []V, c []float64) (float64, float64) {
	s := 0.0
	sq := 0.0
	for idx, e := range eval {
		v := p.GreatCircleDistance(e) * c[idx]
		s += v
		sq += v * v
	}
	return s, sq
}

// geodesicDistancesFromI measures the sum of all great circle distances from an
// evaluation point at index pidx. Everything must lie on the unit sphere.
//
// c is a parallel array of eval, for weights.
//
// Equation 1 for Step 1 of Algorithm
func geodesicDistancesFromI(pidx int, eval []V, c []float64) (float64, float64) {
	s := 0.0
	sq := 0.0
	p := eval[pidx]
	for idx, e := range eval {
		if idx == pidx {
			continue
		}
		v := p.GreatCircleDistance(e) * c[idx]
		s += v
		sq += v * v
	}
	return s, sq
}

// optimalityConditionFromI checks whether an existing location meets the
// conditions for optimality.
//
// c is a parallel array of eval, for weights.
//
// Equation 15
func optimalityConditionFromI(pidx int, eval []V, c []float64, tolerance float64) bool {
	t := c[pidx]
	p := eval[pidx]
	var s V
	for idx, e := range eval {
		if idx == pidx {
			continue
		}
		num := p.Sub(e.DivScalar(p.Dot(e)))
		den := num.Norm()
		s = s.Add(num.MulScalar(c[idx]).DivScalar(den))
	}
	return s.Norm() <= t+tolerance
}

// optimalityConditionFromI checks whether an arbitrary location meets the
// conditions for optimality.
//
// c is a parallel array of eval, for weights.
//
// Equation 16
func optimalityCondition(p V, eval []V, c []float64, tolerance float64) bool {
	var s V
	for idx, e := range eval {
		num := p.Sub(e.DivScalar(p.Dot(e)))
		den := num.Norm()
		s = s.Add(num.MulScalar(c[idx]).DivScalar(den))
	}
	return s.Norm() > -tolerance && s.Norm() < tolerance
}

// ajx produces a point coplanar in the unique plane that is tangent on a sphere
// at x, which corresponds to a different given point aj on the sphere.
//
// x and aj must be points on the unit sphere.
//
// This essentially maps the non-Euclidean multi-facility location problem into
// the Euclidean space.
//
// Equation 9
func ajx(aj, x V) V {
	return aj.DivScalar(x.Dot(aj))
}

// d is the algorithm Step 2 definition of d.
//
// t = target index of "minimum" nonsmooth point
// a = set of all existing locations
// c = set of all weights, parallel array to a.
func d(t int, a []V, c []float64) V {
	at := a[t]
	var s V
	for j, aj := range a {
		if j == t {
			continue
		}
		// stp cannot handle antipoles.
		var ajat V
		if aj.Equals(at.MulScalar(-1)) {
			if len(a) == 2 {
				// Arbitrary direction.
				add := V{1, 0, 0}
				if at.Equals(add) {
					add = V{0, 1, 0}
				}
				ajat = stp(aj, at.Add(add).Unit())
			} else {
				continue
			}
		} else {
			// ajat := ajx(aj, at)
			ajat = stp(aj, at)
		}
		num := at.Sub(ajat)
		denom := num.Norm()
		s = s.Add(num.MulScalar(c[j]).DivScalar(denom))
	}
	return s.MulScalar(-1)
}

// dx is the algorithms Step 3 definition of dk.
//
// x = a smooth search point
// a = set of all existing locations
// c = set of all weights, parallel array to a.
func dx(xk V, a []V, c []float64) V {
	var s V
	for j, aj := range a {
		// stp cannot handle antipoles.
		if aj.Equals(xk.MulScalar(-1)) {
			continue
		}
		// ajxk := ajx(aj, xk)
		ajxk := stp(aj, xk)
		num := xk.Sub(ajxk)
		denom := num.Norm()
		s = s.Add(num.MulScalar(c[j]).DivScalar(denom))
	}
	return s.MulScalar(-1)
}

// alphax is the algorithms Step 3 definition of alphak.
//
// x = a smooth search point
// a = set of all existing locations
// c = set of all weights, parallel array to a.
func alphax(xk V, a []V, c []float64) float64 {
	s := 0.0
	for j, aj := range a {
		// stp cannot handle antipoles.
		if aj.Equals(xk.MulScalar(-1)) {
			continue
		}
		// ajxk := ajx(aj, xk)
		ajxk := stp(aj, xk)
		s += c[j] / xk.Sub(ajxk).Norm()
	}
	return 1 / s
}

// gradFxk is the gradient of F at xk.
func gradFxk(xk V, a []V, c []float64) V {
	var s V
	for j, aj := range a {
		// stp cannot handle antipoles.
		if aj.Equals(xk.MulScalar(-1)) {
			continue
		}
		// ajxk := ajx(aj, xk)
		ajxk := stp(aj, xk)
		num := xk.Sub(ajxk)
		denom := num.Norm()
		s = s.Add(num.MulScalar(c[j]).DivScalar(denom))
	}
	return s
}

// stp is supposed to replace ajx, which is a Gnomonic projection, with a
// stereographic projection that has had projections rescaled to accurately
// reflect distances.
//
// If x == aj, x is returned.
//
// If -x == aj, -x is returned (which is useless, so check for this!).
func stp(aj, x V) V {
	if aj.Equals(x) {
		return x
	}
	// We are projective from the pole onto a plane that goes through the
	// origin, which means we want the anti-pole as that will project the
	// far hemisphere outside, and the hemisphere near x inside.
	p := x.MulScalar(-1)
	if aj.Equals(p) {
		return p
	}
	// Methodology based on the answer at:
	// https://math.stackexchange.com/questions/454507/stereographic-projection-when-the-north-south-pole-is-not-given-by-0-pm/607434#607434
	//
	// First, project aj onto a plane parallel to the plane tangent to the
	// sphere at x but through the origin.
	num := aj.Sub(p)
	denom := 1 - aj.Dot(p)
	proj0 := p.Add(num.DivScalar(denom))
	// Scale the point to actually reflect its great circle distance.
	dist := x.GreatCircleDistance(aj)
	r := proj0.Unit().MulScalar(dist)
	// Translate to be on the parallel plane that passes through x
	return r.Add(x)
}
