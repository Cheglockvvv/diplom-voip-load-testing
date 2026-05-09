package worker

import (
	"context"
	"fmt"
	"math/rand"
	"net"
	"strings"
	"sync"
	"time"

	"diplom_code/internal/config"
	"diplom_code/internal/metrics"
	"diplom_code/internal/qos"
	"diplom_code/internal/rtp"
	"diplom_code/internal/sip"
)

type Runner struct {
	metrics *metrics.Registry
}

func NewRunner(m *metrics.Registry) *Runner {
	return &Runner{metrics: m}
}

func (r *Runner) Run(ctx context.Context, sc config.Scenario) error {
	r.metrics.ScenarioRunning.Set(1)
	defer r.metrics.ScenarioRunning.Set(0)

	duration := time.Duration(sc.DurationSeconds) * time.Second
	endAt := time.Now().Add(duration)
	if sc.CPS <= 0 {
		sc.CPS = 1
	}

	var wg sync.WaitGroup
	for time.Now().Before(endAt) {
		currentCPS := rampedCPS(sc, time.Now(), endAt)
		if currentCPS <= 0 {
			time.Sleep(250 * time.Millisecond)
			continue
		}
		interval := time.Second / time.Duration(currentCPS)
		ticker := time.NewTicker(interval)

		loopCtx, cancel := context.WithDeadline(ctx, time.Now().Add(1*time.Second))
		for {
			select {
			case <-loopCtx.Done():
				cancel()
				ticker.Stop()
				goto nextSecond
			case <-ticker.C:
				wg.Add(1)
				go func() {
					defer wg.Done()
					r.executeOne(ctx, sc)
				}()
			}
		}
	nextSecond:
	}
	wg.Wait()
	return nil
}

func (r *Runner) executeOne(ctx context.Context, sc config.Scenario) {
	switch sc.Mode {
	case "registration_storm":
		r.runRegistration(ctx, sc)
	case "call_setup_rate":
		r.runCallSetup(ctx, sc, false)
	case "media_stress":
		r.runCallSetup(ctx, sc, true)
	default:
		r.runRegistration(ctx, sc)
	}
}

func (r *Runner) runRegistration(ctx context.Context, sc config.Scenario) {
	remote := fmt.Sprintf("%s:%d", sc.Target.Host, sc.Target.SIPPort)
	conn, err := net.Dial("udp", remote)
	if err != nil {
		r.metrics.ObserveSIP("REGISTER", "000", 0)
		return
	}
	defer conn.Close()

	cseq := rand.Intn(100000) + 1
	req := sip.BuildRegister(sc.Target.Host, sc.Target.SIPPort, safeUser(sc), safeDomain(sc), cseq)
	start := time.Now()
	_ = conn.SetDeadline(time.Now().Add(1200 * time.Millisecond))
	if _, err := conn.Write([]byte(req)); err != nil {
		r.metrics.ObserveSIP("REGISTER", "000", time.Since(start))
		return
	}

	buf := make([]byte, 4096)
	n, err := conn.Read(buf)
	if err != nil {
		r.metrics.ObserveSIP("REGISTER", "408", time.Since(start))
		return
	}
	status := sip.ParseStatusCode(string(buf[:n]))
	r.metrics.ObserveSIP("REGISTER", status, time.Since(start))
}

func (r *Runner) runCallSetup(ctx context.Context, sc config.Scenario, withMedia bool) {
	remote := fmt.Sprintf("%s:%d", sc.Target.Host, sc.Target.SIPPort)
	conn, err := net.Dial("udp", remote)
	if err != nil {
		r.metrics.RecordCallAttempt(false, false)
		r.metrics.ObserveSIP("INVITE", "000", 0)
		return
	}
	defer conn.Close()

	fsm := sip.NewFSM()
	fsm.StartCall()

	cseq := rand.Intn(100000) + 1
	fromUser := safeUser(sc)
	toUser := "1001"
	invite := sip.BuildInvite(sc.Target.Host, sc.Target.SIPPort, fromUser, toUser, safeDomain(sc), cseq)
	start := time.Now()
	_ = conn.SetDeadline(time.Now().Add(2 * time.Second))
	if _, err := conn.Write([]byte(invite)); err != nil {
		r.metrics.RecordCallAttempt(false, false)
		r.metrics.ObserveSIP("INVITE", "000", time.Since(start))
		return
	}

	buf := make([]byte, 8192)
	n, err := conn.Read(buf)
	if err != nil {
		r.metrics.RecordCallAttempt(false, false)
		r.metrics.ObserveSIP("INVITE", "408", time.Since(start))
		return
	}
	resp := string(buf[:n])
	status := sip.ParseStatusCode(resp)
	r.metrics.ObserveSIP("INVITE", status, time.Since(start))

	answered := status == "200"
	networkOK := answered || status == "486" || status == "480"
	if strings.HasPrefix(status, "18") {
		fsm.OnProvisional()
		_ = conn.SetDeadline(time.Now().Add(2 * time.Second))
		n2, err2 := conn.Read(buf)
		if err2 == nil {
			resp2 := string(buf[:n2])
			status2 := sip.ParseStatusCode(resp2)
			r.metrics.ObserveSIP("INVITE", status2, time.Since(start))
			status = status2
			answered = status2 == "200"
			networkOK = answered || status2 == "486" || status2 == "480"
			if answered {
				resp = resp2
			}
		}
	}

	if answered {
		fsm.OnAccepted()
		callID := sip.ParseCallID(resp)
		toTag := sip.ParseToTag(resp)
		ack := sip.BuildAck(callID, fromUser, toUser, safeDomain(sc), toTag, cseq)
		_, _ = conn.Write([]byte(ack))
		r.metrics.ActiveCalls.Inc()
		defer r.metrics.ActiveCalls.Dec()

		delayMS := float64(time.Since(start).Milliseconds())
		if withMedia && sc.RTP.Enabled {
			engine := rtp.New(sc.RTP.PacketMS, sc.RTP.PayloadSize)
			remoteRTP := fmt.Sprintf("%s:%d", sc.Target.Host, sc.Target.RTPPort)
			rtpStats, _ := engine.Stream(ctx, remoteRTP, time.Duration(sc.CallDurationSeconds)*time.Second)
			r.metrics.RTPPacketLossPct.Set(rtpStats.LossPct)
			r.metrics.RTPJitterMS.Set(rtpStats.JitterMS)
			r.metrics.RTPMOSEstimated.Set(qos.EstimateMOS(rtpStats.LossPct, rtpStats.JitterMS, delayMS))
		} else {
			time.Sleep(time.Duration(sc.CallDurationSeconds) * time.Second)
		}

		bye := sip.BuildBye(callID, fromUser, toUser, safeDomain(sc), toTag, cseq+1)
		_, _ = conn.Write([]byte(bye))
		fsm.OnTerminated()
	}

	r.metrics.RecordCallAttempt(answered, networkOK)
}

func safeDomain(sc config.Scenario) string {
	if sc.Target.Domain == "" {
		return sc.Target.Host
	}
	return sc.Target.Domain
}

func safeUser(sc config.Scenario) string {
	if sc.Target.Username == "" {
		return "1000"
	}
	return sc.Target.Username
}

func rampedCPS(sc config.Scenario, now time.Time, end time.Time) int {
	start := end.Add(-time.Duration(sc.DurationSeconds) * time.Second)
	target := sc.CPS
	if target <= 0 {
		return 1
	}
	rampUp := time.Duration(sc.RampUpSeconds) * time.Second
	rampDown := time.Duration(sc.RampDownSeconds) * time.Second

	if rampUp > 0 && now.Before(start.Add(rampUp)) {
		progress := float64(now.Sub(start)) / float64(rampUp)
		return max(1, int(progress*float64(target)))
	}
	if rampDown > 0 && now.After(end.Add(-rampDown)) {
		progress := float64(end.Sub(now)) / float64(rampDown)
		return max(1, int(progress*float64(target)))
	}
	return target
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
