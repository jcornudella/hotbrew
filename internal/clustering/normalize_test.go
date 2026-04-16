package clustering

import (
	"testing"

	"github.com/jcornudella/hotbrew/pkg/trss"
)

func TestNormalizeTitleCollapsesCasePunctAndWhitespace(t *testing.T) {
	got := NormalizeTitle("Agents Take Over DevTools!!")
	want := "agents take over devtools"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestNormalizeTitleHandlesUnicode(t *testing.T) {
	got := NormalizeTitle("Café — owned by ACME?")
	want := "café owned by acme"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestCanonicalKeyPrefersCanonicalURL(t *testing.T) {
	item := trss.Item{URL: "https://a.example/X", URLCanonical: "https://A.Example/Y"}
	if got := CanonicalKey(item); got != "https://a.example/y" {
		t.Fatalf("unexpected canonical key: %q", got)
	}
}

func TestCanonicalKeyFallsBackToURLWhenCanonicalMissing(t *testing.T) {
	item := trss.Item{URL: "https://A.Example/Z"}
	if got := CanonicalKey(item); got != "https://a.example/z" {
		t.Fatalf("unexpected canonical key: %q", got)
	}
}

func TestDomainStripsWWW(t *testing.T) {
	item := trss.Item{URL: "https://www.example.com/a/b"}
	if got := Domain(item); got != "example.com" {
		t.Fatalf("expected example.com, got %q", got)
	}
}

func TestRepoSignatureExtractsOwnerRepo(t *testing.T) {
	item := trss.Item{URL: "https://github.com/anthropics/claude-cookbooks/tree/main"}
	if got := RepoSignature(item); got != "github.com/anthropics/claude-cookbooks" {
		t.Fatalf("unexpected repo signature: %q", got)
	}
}

func TestRepoSignatureEmptyForNonGitHub(t *testing.T) {
	item := trss.Item{URL: "https://example.com/anthropics/claude-cookbooks"}
	if got := RepoSignature(item); got != "" {
		t.Fatalf("expected empty, got %q", got)
	}
}
