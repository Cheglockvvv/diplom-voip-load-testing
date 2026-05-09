package config

import "testing"

func TestScenarioValidateSuccess(t *testing.T) {
	s := Scenario{
		Name:             "ok",
		Mode:             "registration_storm",
		Users:            10,
		CPS:              5,
		DurationSeconds:  30,
		SIPTimeoutMS:     1200,
		SIPRetryAttempts: 2,
		Target: TargetConfig{
			Host:    "127.0.0.1",
			SIPPort: 5060,
		},
	}
	if err := s.Validate(); err != nil {
		t.Fatalf("expected valid scenario, got error: %v", err)
	}
}

func TestScenarioValidateFailsOnMissingHost(t *testing.T) {
	s := Scenario{
		Name:            "bad",
		Mode:            "call_setup_rate",
		Users:           10,
		CPS:             5,
		DurationSeconds: 30,
		Target: TargetConfig{
			SIPPort: 5060,
		},
	}
	if err := s.Validate(); err == nil {
		t.Fatal("expected validation error for missing host")
	}
}

func TestScenarioValidateFailsOnUnsupportedMode(t *testing.T) {
	s := Scenario{
		Name:            "bad-mode",
		Mode:            "unknown",
		Users:           10,
		CPS:             5,
		DurationSeconds: 30,
		Target: TargetConfig{
			Host:    "127.0.0.1",
			SIPPort: 5060,
		},
	}
	if err := s.Validate(); err == nil {
		t.Fatal("expected validation error for unsupported mode")
	}
}
