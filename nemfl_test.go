package scr

import (
	"testing"
)

func fWithinTolerance(actual, expected, tolerance float64) bool {
	return actual < expected+tolerance && actual > expected-tolerance
}

func vWithinTolerance(actual, expected V, tolerance float64) bool {
	return fWithinTolerance(actual.X, expected.X, tolerance) &&
		fWithinTolerance(actual.Y, expected.Y, tolerance) &&
		fWithinTolerance(actual.Z, expected.Z, tolerance)
}

func TestPaperExample1(t *testing.T) {
	existing := []V{
		V{11.9472, 68.6294, 71.7445},
		V{64.1042, 13.7732, 75.5046},
		V{64.5830, 26.4982, 71.6022},
		V{31.3250, 48.4404, 81.6840},
		V{1.4133, 70.3890, 71.0168},
		V{52.3136, 44.8641, 72.4603},
		V{67.5622, 11.7916, 72.7757},
		V{42.4400, 55.0978, 71.8546},
		V{4.4998, 69.7835, 71.4843},
		V{42.5885, 55.7987, 71.2231},
		V{56.0900, 41.2539, 71.7777},
		V{7.8076, 67.8472, 73.0465},
		V{34.5160, 60.6224, 71.6490},
		V{42.5769, 55.6421, 71.3524},
		V{49.6205, 50.2590, 70.7943},
		V{48.8773, 50.0174, 71.4791},
		V{61.9993, 33.4040, 70.9948},
		V{10.1102, 68.6413, 72.0150},
		V{60.5060, 35.0758, 71.4753},
		V{4.5250, 68.8010, 72.4289},
	}
	weights := []float64{
		0.0004,
		26.6384,
		33.9648,
		41.5483,
		33.5575,
		20.8743,
		42.3083,
		20.8000,
		13.1226,
		31.6319,
		12.3519,
		32.5759,
		13.6355,
		11.8887,
		24.3259,
		45.2327,
		49.3321,
		47.3882,
		13.8541,
		47.0490,
	}
	starting := V{61.3027, 7.7592, 78.6243}
	for idx := range existing {
		existing[idx] = existing[idx].DivScalar(100)
		// weights[idx] = weights[idx] / 100
	}
	starting = starting.DivScalar(100)
	actual := SolveNonEuclideanMultifacilityLocationSkipNonSmooth(
		existing,
		weights,
		0.0001, 0.0001,
		starting)
	expected := V{43.8601, 51.4813, 73.6612}
	expected = expected.DivScalar(100)
	if !vWithinTolerance(actual, expected, 0.0001) {
		t.Fatalf("expected {%v, %v, %v}, got {%v, %v, %v} for tol=%v", expected.X, expected.Y, expected.Z, actual.X, actual.Y, actual.Z, 0.0001)
	}
}

func TestPaperExample2(t *testing.T) {
	t.Log("No data was given to test against the results. We win.")
	// Woohoo we did it.
}

// Proves the paper can't handle an entire sphere.
func TestThreeOrthogonalPoints(t *testing.T) {
	existing := []V{
		V{0, -1, 0},
		V{0, 0, 1},
		V{1, 0, 0},
	}
	weights := []float64{1, 1, 1}
	actual := SolveNonEuclideanMultifacilityLocation(
		existing,
		weights,
		0.0001, 0.0001)
	expected := V{0.5773042039196745, -0.5774028380579979, 0.5773437613235639}
	if !vWithinTolerance(actual, expected, 0.0001) {
		t.Fatalf("expected {%v, %v, %v}, got {%v, %v, %v} for tol=%v", expected.X, expected.Y, expected.Z, actual.X, actual.Y, actual.Z, 0.0001)
	}
}

func TestStereographicPrjection(t *testing.T) {
	v := V{X: 1, Y: 1, Z: 1}
	v = v.Unit()
	aj := V{X: 1}
	actual := stp(aj, v)
	expected := V{X: 1.3573630215917256, Y: 0.1873438929885758, Z: 0.1873438929885758}
	if !vWithinTolerance(actual, expected, 0.0001) {
		t.Fatalf("expected {%v, %v, %v}, got {%v, %v, %v} for tol=%v", expected.X, expected.Y, expected.Z, actual.X, actual.Y, actual.Z, 0.0001)
	}
}

func TestEqualEverywhere(t *testing.T) {
	existing := []V{
		V{0, 0, -1},
		V{0, -1, 0},
		V{-1, 0, 0},
		V{0, 0, 1},
		V{0, 1, 0},
		V{1, 0, 0},
	}
	weights := []float64{1, 1, 1, 1, 1, 1}
	actual := SolveNonEuclideanMultifacilityLocation(
		existing,
		weights,
		0.0001, 0.0001)
	expected := V{X: 0.396167579913271, Y: 0.7359459057790325, Z: -0.549030848307034}
	if !vWithinTolerance(actual, expected, 0.0001) {
		t.Fatalf("expected {%v, %v, %v}, got {%v, %v, %v} for tol=%v", expected.X, expected.Y, expected.Z, actual.X, actual.Y, actual.Z, 0.0001)
	}

}
