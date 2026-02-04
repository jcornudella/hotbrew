package trss

import (
	"crypto/sha256"
	"fmt"
	"net/url"
	"strings"
)

// UTM and tracking parameters to strip from URLs.
var stripParams = map[string]bool{
	"utm_source":   true,
	"utm_medium":   true,
	"utm_campaign": true,
	"utm_content":  true,
	"utm_term":     true,
	"ref":          true,
	"source":       true,
	"fbclid":       true,
	"gclid":        true,
}

// CanonicalURL normalizes a URL by stripping tracking parameters,
// trailing slashes, and lowercasing the host.
func CanonicalURL(rawURL string) string {
	if rawURL == "" {
		return ""
	}

	u, err := url.Parse(rawURL)
	if err != nil {
		return rawURL
	}

	// Lowercase scheme and host
	u.Scheme = strings.ToLower(u.Scheme)
	u.Host = strings.ToLower(u.Host)

	// Strip tracking params
	q := u.Query()
	for param := range stripParams {
		q.Del(param)
	}
	u.RawQuery = q.Encode()

	// Strip trailing slash (but keep root "/")
	if len(u.Path) > 1 {
		u.Path = strings.TrimRight(u.Path, "/")
	}

	// Strip fragment
	u.Fragment = ""

	return u.String()
}

// Fingerprint generates a SHA-256 hash of a canonical URL.
// Used for deduplication across sources.
func Fingerprint(canonicalURL string) string {
	h := sha256.Sum256([]byte(canonicalURL))
	return fmt.Sprintf("sha256:%x", h)
}

// GenerateID creates a short deterministic ID from a URL or fallback key.
// The ID is the first 12 hex chars of the SHA-256 hash.
func GenerateID(urlOrKey string) string {
	h := sha256.Sum256([]byte(urlOrKey))
	return fmt.Sprintf("sha256:%x", h[:6]) // 12 hex chars
}

// FallbackKey creates a dedup key when no URL is available.
func FallbackKey(title, sourceName string) string {
	return title + "||" + sourceName
}
