package xbookmarks

import (
	"crypto/sha256"
	"encoding/base64"
	"testing"
	"time"
)

func TestNewPKCEPair_VerifierLengthAndChallengeMatch(t *testing.T) {
	verifier, challenge, err := newPKCEPair()
	if err != nil {
		t.Fatalf("newPKCEPair: %v", err)
	}
	// RFC 7636 requires verifier length between 43 and 128.
	if len(verifier) < 43 || len(verifier) > 128 {
		t.Errorf("verifier length out of spec: %d", len(verifier))
	}
	sum := sha256.Sum256([]byte(verifier))
	want := base64.RawURLEncoding.EncodeToString(sum[:])
	if challenge != want {
		t.Errorf("challenge mismatch: got %q want %q", challenge, want)
	}
}

func TestTokenValid(t *testing.T) {
	cases := []struct {
		name string
		tok  *Token
		want bool
	}{
		{"nil", nil, false},
		{"empty access", &Token{ExpiresAt: time.Now().Add(time.Hour)}, false},
		{"expired", &Token{AccessToken: "a", ExpiresAt: time.Now().Add(-time.Second)}, false},
		{"near-expiry", &Token{AccessToken: "a", ExpiresAt: time.Now().Add(10 * time.Second)}, false},
		{"fresh", &Token{AccessToken: "a", ExpiresAt: time.Now().Add(time.Hour)}, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.tok.Valid(); got != tc.want {
				t.Errorf("Valid() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestClientID_EnvOverridesConfig(t *testing.T) {
	t.Setenv("HOTBREW_X_CLIENT_ID", "from-env")
	if got := ClientID("from-config"); got != "from-env" {
		t.Errorf("got %q, want from-env", got)
	}
	t.Setenv("HOTBREW_X_CLIENT_ID", "")
	if got := ClientID("from-config"); got != "from-config" {
		t.Errorf("got %q, want from-config", got)
	}
}

func TestBuildAuthorizeURL_HasRequiredParams(t *testing.T) {
	u := buildAuthorizeURL("client-123", "challenge-abc", "state-xyz")
	for _, needle := range []string{
		"client_id=client-123",
		"code_challenge=challenge-abc",
		"code_challenge_method=S256",
		"state=state-xyz",
		"response_type=code",
		"scope=",
	} {
		if !contains(u, needle) {
			t.Errorf("authorize URL missing %q: %s", needle, u)
		}
	}
}

func contains(haystack, needle string) bool {
	for i := 0; i+len(needle) <= len(haystack); i++ {
		if haystack[i:i+len(needle)] == needle {
			return true
		}
	}
	return false
}
