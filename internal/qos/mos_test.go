package qos

import "testing"

func TestEstimateMOSBoundaries(t *testing.T) {
	cases := []struct {
		name         string
		loss         float64
		jitter       float64
		delay        float64
		minInclusive float64
		maxInclusive float64
	}{
		{
			name:         "excellent_network",
			loss:         0,
			jitter:       1,
			delay:        30,
			minInclusive: 4.0,
			maxInclusive: 4.5,
		},
		{
			name:         "bad_network",
			loss:         20,
			jitter:       80,
			delay:        500,
			minInclusive: 1.0,
			maxInclusive: 2.5,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := EstimateMOS(tc.loss, tc.jitter, tc.delay)
			if got < tc.minInclusive || got > tc.maxInclusive {
				t.Fatalf("mos %f out of range [%f,%f]", got, tc.minInclusive, tc.maxInclusive)
			}
		})
	}
}
