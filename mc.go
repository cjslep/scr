package scr

func MonteCarloMinimizer(
	existingLocations []V, // Existing locations on a unit sphere
	existingLocationWeights []float64) V {
	v := RandomVector()
	min, _ := geodesicDistances(v, existingLocations, existingLocationWeights)
	minV := v
	for i := 0; i < 1000000; i++ {
		v = RandomVector()
		d, _ := geodesicDistances(v, existingLocations, existingLocationWeights)
		if d < min {
			min = d
			minV = v
		}
	}
	return minV
}
