package main

func fit(x, oldmin, oldmax, newmin, newmax float32) float32 {
	if x <= oldmin {
		return newmin
	}
	if x >= oldmax {
		return newmax
	}
	r := (x - oldmin) / (oldmax - oldmin)
	return newmin + (newmax-newmin)*r
}
