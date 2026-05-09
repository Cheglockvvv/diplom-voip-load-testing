package sip

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"strings"
)

type DigestChallenge struct {
	Realm     string
	Nonce     string
	Algorithm string
	QOP       string
}

func ParseDigestChallenge(resp string) (DigestChallenge, error) {
	raw := HeaderValue(resp, "WWW-Authenticate")
	if raw == "" {
		raw = HeaderValue(resp, "Proxy-Authenticate")
	}
	if raw == "" {
		return DigestChallenge{}, fmt.Errorf("digest challenge header not found")
	}
	lower := strings.ToLower(raw)
	if !strings.HasPrefix(lower, "digest ") {
		return DigestChallenge{}, fmt.Errorf("unsupported auth scheme")
	}
	params := parseDigestParams(strings.TrimSpace(raw[len("Digest "):]))
	ch := DigestChallenge{
		Realm:     params["realm"],
		Nonce:     params["nonce"],
		Algorithm: params["algorithm"],
		QOP:       params["qop"],
	}
	if ch.Realm == "" || ch.Nonce == "" {
		return DigestChallenge{}, fmt.Errorf("invalid digest challenge")
	}
	if ch.Algorithm == "" {
		ch.Algorithm = "MD5"
	}
	if strings.Contains(ch.QOP, ",") {
		parts := strings.Split(ch.QOP, ",")
		for _, p := range parts {
			if strings.TrimSpace(p) == "auth" {
				ch.QOP = "auth"
				break
			}
		}
	}
	return ch, nil
}

func BuildDigestAuthorization(username, password, method, uri string, ch DigestChallenge) string {
	ha1 := md5Hex(fmt.Sprintf("%s:%s:%s", username, ch.Realm, password))
	ha2 := md5Hex(fmt.Sprintf("%s:%s", method, uri))
	response := md5Hex(fmt.Sprintf("%s:%s:%s", ha1, ch.Nonce, ha2))
	return fmt.Sprintf(
		"Digest username=\"%s\", realm=\"%s\", nonce=\"%s\", uri=\"%s\", response=\"%s\", algorithm=%s",
		username, ch.Realm, ch.Nonce, uri, response, ch.Algorithm,
	)
}

func parseDigestParams(raw string) map[string]string {
	result := make(map[string]string)
	for _, item := range strings.Split(raw, ",") {
		kv := strings.SplitN(strings.TrimSpace(item), "=", 2)
		if len(kv) != 2 {
			continue
		}
		key := strings.ToLower(strings.TrimSpace(kv[0]))
		value := strings.Trim(strings.TrimSpace(kv[1]), "\"")
		result[key] = value
	}
	return result
}

func md5Hex(v string) string {
	sum := md5.Sum([]byte(v))
	return hex.EncodeToString(sum[:])
}
