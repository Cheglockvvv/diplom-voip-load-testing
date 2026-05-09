package worker

import (
	"testing"
	"time"

	"diplom_code/internal/config"
)

func TestRampedCPSRampUp(t *testing.T) {
	sc := config.Scenario{
		CPS:             100,
		DurationSeconds: 60,
		RampUpSeconds:   20,
		RampDownSeconds: 20,
	}
	end := time.Now().Add(60 * time.Second)
	now := end.Add(-60 * time.Second).Add(10 * time.Second)
	got := rampedCPS(sc, now, end)
	if got < 45 || got > 55 {
		t.Fatalf("expected around 50 cps in middle of ramp-up, got %d", got)
	}
}

func TestRampedCPSSteady(t *testing.T) {
	sc := config.Scenario{
		CPS:             100,
		DurationSeconds: 60,
		RampUpSeconds:   10,
		RampDownSeconds: 10,
	}
	end := time.Now().Add(60 * time.Second)
	now := end.Add(-30 * time.Second)
	got := rampedCPS(sc, now, end)
	if got != 100 {
		t.Fatalf("expected steady cps 100, got %d", got)
	}
}

func TestRampedCPSRampDown(t *testing.T) {
	sc := config.Scenario{
		CPS:             100,
		DurationSeconds: 60,
		RampUpSeconds:   10,
		RampDownSeconds: 20,
	}
	end := time.Now().Add(60 * time.Second)
	now := end.Add(-10 * time.Second)
	got := rampedCPS(sc, now, end)
	if got < 45 || got > 55 {
		t.Fatalf("expected around 50 cps in middle of ramp-down, got %d", got)
	}
}
