package sip

import (
	"strings"
	"testing"
)

func TestParseDigestChallenge(t *testing.T) {
	resp := strings.Join([]string{
		"SIP/2.0 401 Unauthorized",
		"WWW-Authenticate: Digest realm=\"asterisk\", nonce=\"abc123\", algorithm=MD5, qop=\"auth\"",
		"",
		"",
	}, "\r\n")
	ch, err := ParseDigestChallenge(resp)
	if err != nil {
		t.Fatalf("unexpected parse error: %v", err)
	}
	if ch.Realm != "asterisk" || ch.Nonce != "abc123" {
		t.Fatalf("unexpected challenge: %+v", ch)
	}
}

func TestBuildDigestAuthorization(t *testing.T) {
	ch := DigestChallenge{Realm: "asterisk", Nonce: "abc123", Algorithm: "MD5"}
	auth := BuildDigestAuthorization("1000", "1000", "REGISTER", "sip:asterisk", ch)
	if !strings.Contains(auth, "Digest username=\"1000\"") {
		t.Fatalf("auth header missing username: %s", auth)
	}
	if !strings.Contains(auth, "response=\"") {
		t.Fatalf("auth header missing response hash: %s", auth)
	}
}

func TestBuildDigestAuthorizationWithQOP(t *testing.T) {
	ch := DigestChallenge{Realm: "asterisk", Nonce: "abc123", Algorithm: "MD5", QOP: "auth"}
	auth := BuildDigestAuthorization("1000", "1000", "REGISTER", "sip:asterisk", ch)
	if !strings.Contains(auth, "qop=auth") {
		t.Fatalf("auth header missing qop: %s", auth)
	}
	if !strings.Contains(auth, "cnonce=") {
		t.Fatalf("auth header missing cnonce: %s", auth)
	}
	if !strings.Contains(auth, "nc=00000001") {
		t.Fatalf("auth header missing nonce count: %s", auth)
	}
}
