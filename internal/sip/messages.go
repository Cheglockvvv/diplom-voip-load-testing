package sip

import (
	"fmt"
	"math/rand"
	"strings"
	"time"
)

func BuildRegister(host string, port int, user, domain string, cseq int) string {
	return buildRegister(host, port, user, domain, cseq, "")
}

func BuildRegisterWithAuthorization(host string, port int, user, domain string, cseq int, authorization string) string {
	return buildRegister(host, port, user, domain, cseq, authorization)
}

func buildRegister(host string, port int, user, domain string, cseq int, authorization string) string {
	branch := randomToken()
	callID := fmt.Sprintf("%s@%s", randomToken(), host)
	tag := randomToken()
	uri := fmt.Sprintf("sip:%s@%s", user, domain)
	headers := []string{
		fmt.Sprintf("REGISTER sip:%s SIP/2.0", domain),
		fmt.Sprintf("Via: SIP/2.0/UDP tester;branch=z9hG4bK-%s", branch),
		"Max-Forwards: 70",
		fmt.Sprintf("From: <sip:%s>;tag=%s", uri, tag),
		fmt.Sprintf("To: <sip:%s>", uri),
		fmt.Sprintf("Call-ID: %s", callID),
		fmt.Sprintf("CSeq: %d REGISTER", cseq),
		fmt.Sprintf("Contact: <sip:%s@tester:%d>", user, port),
		"Expires: 300",
	}
	if authorization != "" {
		headers = append(headers, fmt.Sprintf("Authorization: %s", authorization))
	}
	headers = append(headers, "Content-Length: 0", "", "")
	return strings.Join(headers, "\r\n")
}

func BuildInvite(host string, port int, fromUser, toUser, domain string, cseq int) string {
	branch := randomToken()
	callID := fmt.Sprintf("%s@%s", randomToken(), host)
	fromTag := randomToken()
	return strings.Join([]string{
		fmt.Sprintf("INVITE sip:%s@%s SIP/2.0", toUser, domain),
		fmt.Sprintf("Via: SIP/2.0/UDP tester;branch=z9hG4bK-%s", branch),
		"Max-Forwards: 70",
		fmt.Sprintf("From: <sip:%s@%s>;tag=%s", fromUser, domain, fromTag),
		fmt.Sprintf("To: <sip:%s@%s>", toUser, domain),
		fmt.Sprintf("Call-ID: %s", callID),
		fmt.Sprintf("CSeq: %d INVITE", cseq),
		fmt.Sprintf("Contact: <sip:%s@tester:%d>", fromUser, port),
		"Content-Type: application/sdp",
		"Content-Length: 0",
		"",
		"",
	}, "\r\n")
}

func BuildAck(callID, fromUser, toUser, domain, toTag string, cseq int) string {
	return strings.Join([]string{
		fmt.Sprintf("ACK sip:%s@%s SIP/2.0", toUser, domain),
		fmt.Sprintf("Via: SIP/2.0/UDP tester;branch=z9hG4bK-%s", randomToken()),
		"Max-Forwards: 70",
		fmt.Sprintf("From: <sip:%s@%s>;tag=%s", fromUser, domain, randomToken()),
		fmt.Sprintf("To: <sip:%s@%s>;tag=%s", toUser, domain, toTag),
		fmt.Sprintf("Call-ID: %s", callID),
		fmt.Sprintf("CSeq: %d ACK", cseq),
		"Content-Length: 0",
		"",
		"",
	}, "\r\n")
}

func BuildBye(callID, fromUser, toUser, domain, toTag string, cseq int) string {
	return strings.Join([]string{
		fmt.Sprintf("BYE sip:%s@%s SIP/2.0", toUser, domain),
		fmt.Sprintf("Via: SIP/2.0/UDP tester;branch=z9hG4bK-%s", randomToken()),
		"Max-Forwards: 70",
		fmt.Sprintf("From: <sip:%s@%s>;tag=%s", fromUser, domain, randomToken()),
		fmt.Sprintf("To: <sip:%s@%s>;tag=%s", toUser, domain, toTag),
		fmt.Sprintf("Call-ID: %s", callID),
		fmt.Sprintf("CSeq: %d BYE", cseq),
		"Content-Length: 0",
		"",
		"",
	}, "\r\n")
}

func ParseStatusCode(resp string) string {
	lines := strings.Split(resp, "\r\n")
	if len(lines) == 0 {
		return "000"
	}
	parts := strings.Split(lines[0], " ")
	if len(parts) < 2 {
		return "000"
	}
	return parts[1]
}

func ParseToTag(resp string) string {
	for _, line := range strings.Split(resp, "\r\n") {
		if strings.HasPrefix(strings.ToLower(line), "to:") {
			parts := strings.Split(line, "tag=")
			if len(parts) == 2 {
				return strings.TrimSpace(parts[1])
			}
		}
	}
	return randomToken()
}

func ParseCallID(resp string) string {
	for _, line := range strings.Split(resp, "\r\n") {
		if strings.HasPrefix(strings.ToLower(line), "call-id:") {
			return strings.TrimSpace(strings.TrimPrefix(line, "Call-ID:"))
		}
	}
	return randomToken()
}

func HeaderValue(resp, headerName string) string {
	prefix := strings.ToLower(headerName) + ":"
	for _, line := range strings.Split(resp, "\r\n") {
		if strings.HasPrefix(strings.ToLower(line), prefix) {
			return strings.TrimSpace(line[len(prefix):])
		}
	}
	return ""
}

func randomToken() string {
	return fmt.Sprintf("%x%x", time.Now().UnixNano(), rand.Int63())
}
