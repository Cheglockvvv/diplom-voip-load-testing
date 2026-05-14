package worker

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"net"
	"strconv"
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
	if err := sc.Validate(); err != nil {
		return err
	}
	r.metrics.ScenarioRunning.Set(1)
	defer r.metrics.ScenarioRunning.Set(0)

	startAt := time.Now()
	endAt := startAt.Add(time.Duration(sc.DurationSeconds) * time.Second)
	maxConcurrent := sc.Users
	if maxConcurrent < 1 {
		maxConcurrent = 1
	}

	var wg sync.WaitGroup
	sem := make(chan struct{}, maxConcurrent)
	for time.Now().Before(endAt) {
		select {
		case <-ctx.Done():
			wg.Wait()
			return nil
		default:
		}

		currentCPS := rampedCPS(sc, time.Now(), endAt)
		if currentCPS <= 0 {
			time.Sleep(250 * time.Millisecond)
			continue
		}
		interval := time.Second / time.Duration(currentCPS)
		ticker := time.NewTicker(interval)
		secondDeadline := time.Now().Add(1 * time.Second)
		for {
			select {
			case <-ctx.Done():
				ticker.Stop()
				wg.Wait()
				return nil
			case <-time.After(time.Until(secondDeadline)):
				ticker.Stop()
				goto nextSecond
			case <-ticker.C:
				select {
				case sem <- struct{}{}:
				case <-ctx.Done():
					ticker.Stop()
					wg.Wait()
					return nil
				}
				wg.Add(1)
				go func() {
					defer wg.Done()
					defer func() { <-sem }()
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
	select {
	case <-ctx.Done():
		return
	default:
	}

	remote := net.JoinHostPort(sc.Target.Host, strconv.Itoa(sc.Target.SIPPort))
	conn, err := net.Dial("udp", remote)
	if err != nil {
		r.metrics.ObserveSIP("REGISTER", "000", 0)
		return
	}
	defer conn.Close()

	cseq := rand.Intn(100000) + 1
	req := sip.BuildRegister(sc.Target.Host, sc.Target.SIPPort, safeUser(sc), safeDomain(sc), cseq)
	start := time.Now()
	sipTimeout := time.Duration(sc.SIPTimeoutMS) * time.Millisecond
	resp, status := r.udpTransaction(ctx, conn, req, "REGISTER", sc.SIPRetryAttempts, sipTimeout)
	r.metrics.ObserveSIP("REGISTER", status, time.Since(start))
	if status != "401" || sc.Target.Password == "" {
		return
	}

	challenge, err := sip.ParseDigestChallenge(resp)
	if err != nil {
		r.metrics.ObserveSIP("REGISTER", "401", time.Since(start))
		return
	}
	uri := fmt.Sprintf("sip:%s", safeDomain(sc))
	authHeader := sip.BuildDigestAuthorization(safeUser(sc), sc.Target.Password, "REGISTER", uri, challenge)
	authReq := sip.BuildRegisterWithAuthorization(sc.Target.Host, sc.Target.SIPPort, safeUser(sc), safeDomain(sc), cseq+1, authHeader)
	authStart := time.Now()
	_, authStatus := r.udpTransaction(ctx, conn, authReq, "REGISTER", sc.SIPRetryAttempts, sipTimeout)
	r.metrics.ObserveSIP("REGISTER", authStatus, time.Since(authStart))
}

func (r *Runner) runCallSetup(ctx context.Context, sc config.Scenario, withMedia bool) {
	select {
	case <-ctx.Done():
		return
	default:
	}

	remote := net.JoinHostPort(sc.Target.Host, strconv.Itoa(sc.Target.SIPPort))
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
	sipTimeout := time.Duration(sc.SIPTimeoutMS) * time.Millisecond
	_ = conn.SetDeadline(time.Now().Add(sipTimeout))
	if err := contextErr(ctx); err != nil {
		r.metrics.RecordCallAttempt(false, false)
		r.metrics.ObserveSIP("INVITE", "499", time.Since(start))
		return
	}

	resp, status := r.udpTransaction(ctx, conn, invite, "INVITE", sc.SIPRetryAttempts, sipTimeout)
	if status == "408" || status == "000" || status == "499" {
		r.metrics.RecordCallAttempt(false, false)
		r.metrics.ObserveSIP("INVITE", status, time.Since(start))
		return
	}
	r.metrics.ObserveSIP("INVITE", status, time.Since(start))

	answered := status == "200"
	networkOK := answered || status == "486" || status == "480"
	if strings.HasPrefix(status, "18") {
		fsm.OnProvisional()
		resp2, status2 := r.readInviteUntilFinal(ctx, conn, sipTimeout, start)
		if status2 != "" {
			status = status2
			answered = status2 == "200"
			networkOK = answered || status2 == "486" || status2 == "480"
			if resp2 != "" {
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
			select {
			case <-ctx.Done():
				return
			case <-time.After(time.Duration(sc.CallDurationSeconds) * time.Second):
			}
		}

		bye := sip.BuildBye(callID, fromUser, toUser, safeDomain(sc), toTag, cseq+1)
		_, byeStatus := r.udpTransaction(ctx, conn, bye, "BYE", sc.SIPRetryAttempts, sipTimeout)
		r.metrics.ObserveSIP("BYE", byeStatus, time.Since(start))
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

func contextErr(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return errors.New("context cancelled")
	default:
		return nil
	}
}

func (r *Runner) udpTransaction(ctx context.Context, conn net.Conn, request string, method string, attempts int, timeout time.Duration) (string, string) {
	if attempts < 1 {
		attempts = 1
	}
	buf := make([]byte, 8192)
	backoff := timeout
	for i := 0; i < attempts; i++ {
		if i > 0 {
			r.metrics.ObserveRetry(method)
		}
		if err := contextErr(ctx); err != nil {
			return "", "499"
		}
		_ = conn.SetDeadline(time.Now().Add(backoff))
		if _, err := conn.Write([]byte(request)); err != nil {
			backoff = minDuration(backoff*2, 3*time.Second)
			continue
		}
		n, err := conn.Read(buf)
		if err != nil {
			backoff = minDuration(backoff*2, 3*time.Second)
			continue
		}
		resp := string(buf[:n])
		return resp, sip.ParseStatusCode(resp)
	}
	return "", "408"
}

func minDuration(a, b time.Duration) time.Duration {
	if a < b {
		return a
	}
	return b
}

func (r *Runner) readInviteUntilFinal(ctx context.Context, conn net.Conn, timeout time.Duration, start time.Time) (string, string) {
	deadline := time.Now().Add(timeout)
	buf := make([]byte, 8192)
	for {
		if err := contextErr(ctx); err != nil {
			return "", "499"
		}
		remaining := time.Until(deadline)
		if remaining <= 0 {
			return "", "408"
		}
		_ = conn.SetDeadline(time.Now().Add(remaining))
		n, err := conn.Read(buf)
		if err != nil {
			return "", "408"
		}
		resp := string(buf[:n])
		status := sip.ParseStatusCode(resp)
		r.metrics.ObserveSIP("INVITE", status, time.Since(start))
		if !strings.HasPrefix(status, "1") {
			return resp, status
		}
	}
}
