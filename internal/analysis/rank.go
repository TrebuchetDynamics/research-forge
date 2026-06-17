package analysis

func kendallTau(xs, ys []float64) float64 {
	concordant := 0.0
	discordant := 0.0
	for i := 0; i < len(xs); i++ {
		for j := i + 1; j < len(xs); j++ {
			dx := xs[i] - xs[j]
			dy := ys[i] - ys[j]
			prod := dx * dy
			if prod > 0 {
				concordant++
			} else if prod < 0 {
				discordant++
			}
		}
	}
	denom := concordant + discordant
	if denom == 0 {
		return 0
	}
	return (concordant - discordant) / denom
}
