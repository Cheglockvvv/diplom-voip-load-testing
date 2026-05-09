package config

import (
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

type Scenario struct {
	Name                string       `yaml:"name" json:"name"`
	Mode                string       `yaml:"mode" json:"mode"`
	Users               int          `yaml:"users" json:"users"`
	CPS                 int          `yaml:"cps" json:"cps"`
	DurationSeconds     int          `yaml:"duration_seconds" json:"duration_seconds"`
	RampUpSeconds       int          `yaml:"ramp_up_seconds" json:"ramp_up_seconds"`
	RampDownSeconds     int          `yaml:"ramp_down_seconds" json:"ramp_down_seconds"`
	CallDurationSeconds int          `yaml:"call_duration_seconds" json:"call_duration_seconds"`
	SIPTimeoutMS        int          `yaml:"sip_timeout_ms" json:"sip_timeout_ms"`
	SIPRetryAttempts    int          `yaml:"sip_retry_attempts" json:"sip_retry_attempts"`
	Target              TargetConfig `yaml:"target" json:"target"`
	RTP                 RTPConfig    `yaml:"rtp" json:"rtp"`
}

type TargetConfig struct {
	Host     string `yaml:"host" json:"host"`
	SIPPort  int    `yaml:"sip_port" json:"sip_port"`
	RTPPort  int    `yaml:"rtp_port" json:"rtp_port"`
	Username string `yaml:"username" json:"username"`
	Password string `yaml:"password" json:"password"`
	Domain   string `yaml:"domain" json:"domain"`
}

type RTPConfig struct {
	Enabled     bool   `yaml:"enabled" json:"enabled"`
	PacketMS    int    `yaml:"packet_ms" json:"packet_ms"`
	PayloadSize int    `yaml:"payload_size" json:"payload_size"`
	Codec       string `yaml:"codec" json:"codec"`
}

func LoadScenario(path string) (Scenario, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return Scenario{}, fmt.Errorf("read scenario: %w", err)
	}
	var cfg Scenario
	if err := yaml.Unmarshal(raw, &cfg); err != nil {
		return Scenario{}, fmt.Errorf("unmarshal scenario: %w", err)
	}
	cfg.fillDefaults()
	return cfg, nil
}

func (s *Scenario) fillDefaults() {
	if s.Mode == "" {
		s.Mode = "registration_storm"
	}
	if s.Users <= 0 {
		s.Users = 100
	}
	if s.CPS <= 0 {
		s.CPS = 10
	}
	if s.DurationSeconds <= 0 {
		s.DurationSeconds = 60
	}
	if s.CallDurationSeconds <= 0 {
		s.CallDurationSeconds = 20
	}
	if s.SIPTimeoutMS <= 0 {
		s.SIPTimeoutMS = 1200
	}
	if s.SIPRetryAttempts <= 0 {
		s.SIPRetryAttempts = 2
	}
	if s.Target.SIPPort == 0 {
		s.Target.SIPPort = 5060
	}
	if s.RTP.PacketMS == 0 {
		s.RTP.PacketMS = 20
	}
	if s.RTP.PayloadSize == 0 {
		s.RTP.PayloadSize = 160
	}
}

func (s Scenario) Validate() error {
	if strings.TrimSpace(s.Name) == "" {
		return fmt.Errorf("scenario name is required")
	}
	switch s.Mode {
	case "registration_storm", "call_setup_rate", "media_stress":
	default:
		return fmt.Errorf("unsupported mode %q", s.Mode)
	}
	if s.DurationSeconds <= 0 {
		return fmt.Errorf("duration_seconds must be > 0")
	}
	if s.CPS <= 0 {
		return fmt.Errorf("cps must be > 0")
	}
	if s.Users <= 0 {
		return fmt.Errorf("users must be > 0")
	}
	if strings.TrimSpace(s.Target.Host) == "" {
		return fmt.Errorf("target.host is required")
	}
	if s.Target.SIPPort <= 0 {
		return fmt.Errorf("target.sip_port must be > 0")
	}
	if s.SIPTimeoutMS <= 0 {
		return fmt.Errorf("sip_timeout_ms must be > 0")
	}
	if s.SIPRetryAttempts <= 0 {
		return fmt.Errorf("sip_retry_attempts must be > 0")
	}
	if s.RTP.Enabled && s.Target.RTPPort <= 0 {
		return fmt.Errorf("target.rtp_port must be > 0 when rtp.enabled=true")
	}
	return nil
}
