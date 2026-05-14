package worker

import (
	"context"
	"fmt"
	"net"
	"strings"
	"testing"
	"time"

	"diplom_code/internal/config"
	"diplom_code/internal/metrics"
)

func TestRunRegistrationDigestFlow(t *testing.T) {
	addr, done := startUDPServer(t, func(conn *net.UDPConn) {
		buf := make([]byte, 8192)
		n, client, err := conn.ReadFromUDP(buf)
		if err != nil {
			t.Errorf("read first register failed: %v", err)
			return
		}
		req1 := string(buf[:n])
		if !strings.Contains(req1, "REGISTER ") {
			t.Errorf("first packet is not REGISTER: %s", req1)
			return
		}
		if strings.Contains(strings.ToLower(req1), "authorization:") {
			t.Errorf("first REGISTER must not have authorization header")
			return
		}
		_, _ = conn.WriteToUDP([]byte(strings.Join([]string{
			"SIP/2.0 401 Unauthorized",
			"Via: SIP/2.0/UDP tester;branch=z9hG4bK-test",
			"Call-ID: reg-test-call",
			"CSeq: 1 REGISTER",
			"WWW-Authenticate: Digest realm=\"asterisk\", nonce=\"abc123\", algorithm=MD5, qop=\"auth\"",
			"Content-Length: 0",
			"",
			"",
		}, "\r\n")), client)

		n, _, err = conn.ReadFromUDP(buf)
		if err != nil {
			t.Errorf("read second register failed: %v", err)
			return
		}
		req2 := string(buf[:n])
		if !strings.Contains(strings.ToLower(req2), "authorization: digest") {
			t.Errorf("second REGISTER must contain digest auth: %s", req2)
			return
		}
		_, _ = conn.WriteToUDP([]byte(strings.Join([]string{
			"SIP/2.0 200 OK",
			"Call-ID: reg-test-call",
			"CSeq: 2 REGISTER",
			"Content-Length: 0",
			"",
			"",
		}, "\r\n")), client)
	})
	defer done()

	r := NewRunner(metrics.NewRegistry("test_worker"))
	sc := config.Scenario{
		Name:             "reg-auth",
		Mode:             "registration_storm",
		Users:            1,
		CPS:              1,
		DurationSeconds:  1,
		SIPTimeoutMS:     600,
		SIPRetryAttempts: 2,
		Target: config.TargetConfig{
			Host:     addr.IP.String(),
			SIPPort:  addr.Port,
			Username: "1000",
			Password: "1000",
			Domain:   "asterisk",
		},
	}
	r.runRegistration(context.Background(), sc)
}

func TestRunCallSetupHandlesProvisionalAndBye(t *testing.T) {
	callID := "call-setup-test"
	toTag := "remote-tag-1"
	gotACK := false
	gotBYE := false

	addr, done := startUDPServer(t, func(conn *net.UDPConn) {
		buf := make([]byte, 8192)
		n, client, err := conn.ReadFromUDP(buf)
		if err != nil {
			t.Errorf("read invite failed: %v", err)
			return
		}
		req := string(buf[:n])
		if !strings.Contains(req, "INVITE ") {
			t.Errorf("expected INVITE request, got: %s", req)
			return
		}

		_, _ = conn.WriteToUDP([]byte(strings.Join([]string{
			"SIP/2.0 180 Ringing",
			"To: <sip:1001@asterisk>;tag=" + toTag,
			"Call-ID: " + callID,
			"CSeq: 1 INVITE",
			"Content-Length: 0",
			"",
			"",
		}, "\r\n")), client)
		_, _ = conn.WriteToUDP([]byte(strings.Join([]string{
			"SIP/2.0 183 Session Progress",
			"To: <sip:1001@asterisk>;tag=" + toTag,
			"Call-ID: " + callID,
			"CSeq: 1 INVITE",
			"Content-Length: 0",
			"",
			"",
		}, "\r\n")), client)
		_, _ = conn.WriteToUDP([]byte(strings.Join([]string{
			"SIP/2.0 200 OK",
			"To: <sip:1001@asterisk>;tag=" + toTag,
			"Call-ID: " + callID,
			"CSeq: 1 INVITE",
			"Content-Length: 0",
			"",
			"",
		}, "\r\n")), client)

		n, _, err = conn.ReadFromUDP(buf)
		if err == nil && strings.Contains(string(buf[:n]), "ACK ") {
			gotACK = true
		}
		n, _, err = conn.ReadFromUDP(buf)
		if err == nil && strings.Contains(string(buf[:n]), "BYE ") {
			gotBYE = true
			_, _ = conn.WriteToUDP([]byte(strings.Join([]string{
				"SIP/2.0 200 OK",
				"Call-ID: " + callID,
				"CSeq: 2 BYE",
				"Content-Length: 0",
				"",
				"",
			}, "\r\n")), client)
		}
	})
	defer done()

	r := NewRunner(metrics.NewRegistry("test_worker"))
	sc := config.Scenario{
		Name:                "call-setup",
		Mode:                "call_setup_rate",
		Users:               1,
		CPS:                 1,
		DurationSeconds:     1,
		SIPTimeoutMS:        800,
		SIPRetryAttempts:    2,
		CallDurationSeconds: 0,
		Target: config.TargetConfig{
			Host:    addr.IP.String(),
			SIPPort: addr.Port,
			Domain:  "asterisk",
		},
	}
	r.runCallSetup(context.Background(), sc, false)
	if !gotACK {
		t.Fatal("expected ACK to be sent after 200 OK")
	}
	if !gotBYE {
		t.Fatal("expected BYE to be sent after call duration")
	}
}

func startUDPServer(t *testing.T, handler func(conn *net.UDPConn)) (*net.UDPAddr, func()) {
	t.Helper()
	addr, err := net.ResolveUDPAddr("udp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("resolve udp addr: %v", err)
	}
	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		t.Fatalf("listen udp: %v", err)
	}
	done := make(chan struct{})
	go func() {
		defer close(done)
		handler(conn)
	}()
	return conn.LocalAddr().(*net.UDPAddr), func() {
		_ = conn.SetDeadline(time.Now().Add(5 * time.Millisecond))
		_ = conn.Close()
		select {
		case <-done:
		case <-time.After(200 * time.Millisecond):
			t.Logf("udp test server did not exit in time: %s", fmt.Sprint(conn.LocalAddr()))
		}
	}
}
