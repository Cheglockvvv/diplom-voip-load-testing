package qos

import "math"

func EstimateMOS(packetLossPct float64, jitterMS float64, oneWayDelayMS float64) float64 {
	if packetLossPct < 0 {
		packetLossPct = 0
	}
	if jitterMS < 0 {
		jitterMS = 0
	}
	if oneWayDelayMS < 0 {
		oneWayDelayMS = 0
	}

	id := 0.024*oneWayDelayMS + 0.11*math.Max(0, oneWayDelayMS-177.3)
	ie := 30*math.Log10(1+packetLossPct)
	if jitterMS > 30 {
		ie += (jitterMS - 30) * 0.2
	}

	r := 94.2 - id - ie
	if r < 0 {
		r = 0
	}
	if r > 100 {
		r = 100
	}

	mos := 1 + 0.035*r + (r*(r-60)*(100-r))*7e-6
	if mos < 1 {
		mos = 1
	}
	if mos > 4.5 {
		mos = 4.5
	}
	return mos
}
