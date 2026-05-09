package metrics

import (
	"net/http"
	"sync/atomic"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type Registry struct {
	reg *prometheus.Registry

	SIPRequestsTotal *prometheus.CounterVec
	SIPRetriesTotal  *prometheus.CounterVec
	RRDSeconds       prometheus.Histogram
	SRDSeconds       prometheus.Histogram
	ActiveCalls      prometheus.Gauge
	RTPPacketLossPct prometheus.Gauge
	RTPJitterMS      prometheus.Gauge
	RTPMOSEstimated  prometheus.Gauge
	ScenarioRunning  prometheus.Gauge

	callAttempts uint64
	callAnswered uint64
	callNetOK    uint64
}

func NewRegistry(component string) *Registry {
	r := &Registry{reg: prometheus.NewRegistry()}
	r.SIPRequestsTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "voip",
		Subsystem: component,
		Name:      "sip_requests_total",
		Help:      "Number of SIP requests by method and status.",
	}, []string{"method", "status"})
	r.SIPRetriesTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "voip",
		Subsystem: component,
		Name:      "sip_retries_total",
		Help:      "Number of SIP request retries.",
	}, []string{"method"})
	r.RRDSeconds = prometheus.NewHistogram(prometheus.HistogramOpts{
		Namespace: "voip",
		Subsystem: component,
		Name:      "registration_delay_seconds",
		Help:      "Registration request delay.",
		Buckets:   []float64{0.01, 0.03, 0.05, 0.1, 0.2, 0.5, 1},
	})
	r.SRDSeconds = prometheus.NewHistogram(prometheus.HistogramOpts{
		Namespace: "voip",
		Subsystem: component,
		Name:      "session_request_delay_seconds",
		Help:      "Session request delay (INVITE to first response).",
		Buckets:   []float64{0.01, 0.03, 0.05, 0.1, 0.2, 0.5, 1, 2},
	})
	r.ActiveCalls = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: "voip",
		Subsystem: component,
		Name:      "active_calls_current",
		Help:      "Current active calls.",
	})
	r.RTPPacketLossPct = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: "voip",
		Subsystem: component,
		Name:      "rtp_packet_loss_pct",
		Help:      "Current RTP packet loss percent.",
	})
	r.RTPJitterMS = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: "voip",
		Subsystem: component,
		Name:      "rtp_jitter_ms",
		Help:      "Current RTP jitter estimate in milliseconds.",
	})
	r.RTPMOSEstimated = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: "voip",
		Subsystem: component,
		Name:      "rtp_mos_estimated",
		Help:      "Estimated MOS score from E-model approximation.",
	})
	r.ScenarioRunning = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: "voip",
		Subsystem: component,
		Name:      "scenario_running",
		Help:      "1 if scenario is currently running.",
	})

	r.reg.MustRegister(
		r.SIPRequestsTotal,
		r.SIPRetriesTotal,
		r.RRDSeconds,
		r.SRDSeconds,
		r.ActiveCalls,
		r.RTPPacketLossPct,
		r.RTPJitterMS,
		r.RTPMOSEstimated,
		r.ScenarioRunning,
		prometheus.NewGaugeFunc(prometheus.GaugeOpts{
			Namespace: "voip",
			Subsystem: component,
			Name:      "asr_ratio",
			Help:      "Answer Seizure Ratio.",
		}, func() float64 {
			attempts := atomic.LoadUint64(&r.callAttempts)
			if attempts == 0 {
				return 1
			}
			return float64(atomic.LoadUint64(&r.callAnswered)) / float64(attempts)
		}),
		prometheus.NewGaugeFunc(prometheus.GaugeOpts{
			Namespace: "voip",
			Subsystem: component,
			Name:      "ner_ratio",
			Help:      "Network Effectiveness Ratio.",
		}, func() float64 {
			attempts := atomic.LoadUint64(&r.callAttempts)
			if attempts == 0 {
				return 1
			}
			return float64(atomic.LoadUint64(&r.callNetOK)) / float64(attempts)
		}),
	)

	return r
}

func (r *Registry) ObserveRetry(method string) {
	r.SIPRetriesTotal.WithLabelValues(method).Inc()
}

func (r *Registry) Handler() http.Handler {
	return promhttp.HandlerFor(r.reg, promhttp.HandlerOpts{})
}

func (r *Registry) ObserveSIP(method, status string, duration time.Duration) {
	r.SIPRequestsTotal.WithLabelValues(method, status).Inc()
	switch method {
	case "REGISTER":
		r.RRDSeconds.Observe(duration.Seconds())
	case "INVITE":
		r.SRDSeconds.Observe(duration.Seconds())
	}
}

func (r *Registry) RecordCallAttempt(answered, networkOK bool) {
	atomic.AddUint64(&r.callAttempts, 1)
	if answered {
		atomic.AddUint64(&r.callAnswered, 1)
	}
	if networkOK {
		atomic.AddUint64(&r.callNetOK, 1)
	}
}
