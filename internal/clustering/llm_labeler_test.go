package clustering

import (
	"context"
	"errors"
	"testing"

	"github.com/jcornudella/hotbrew/pkg/trss"
)

// stubLabeler is a deterministic fake Labeler for testing the
// ClusterWith integration. Returns whatever slug the test plants in
// the slugForID map; missing IDs are not labeled (caller falls back
// to keyword).
type stubLabeler struct {
	slugForID map[string]string
	calls     int
	err       error
}

func (s *stubLabeler) LabelClusters(ctx context.Context, batches []ClusterBatch) ([]ClusterLabel, error) {
	s.calls++
	if s.err != nil {
		return nil, s.err
	}
	out := make([]ClusterLabel, 0, len(batches))
	for _, b := range batches {
		if slug, ok := s.slugForID[b.Items[0].ID]; ok {
			out = append(out, ClusterLabel{ID: b.ID, Slug: slug})
		}
	}
	return out, nil
}

func TestClusterWithUsesLabelerSlug(t *testing.T) {
	items := []trss.Item{
		// Title with no AI keywords — keyword matcher would call this "general".
		{ID: "x-1", Title: "DeepSeek v4 people", Source: trss.ItemSource{Name: "Reddit AI"}},
	}

	labeler := &stubLabeler{
		slugForID: map[string]string{"x-1": "ai"},
	}

	clusters := ClusterWith(context.Background(), items, labeler)
	if len(clusters) != 1 {
		t.Fatalf("expected 1 cluster, got %d", len(clusters))
	}
	if clusters[0].Slug != "ai" {
		t.Errorf("labeler slug should win over keyword fallback: got %q want ai", clusters[0].Slug)
	}
	if labeler.calls != 1 {
		t.Errorf("labeler should be called exactly once for batched clusters, got %d", labeler.calls)
	}
}

func TestClusterWithFallsBackToKeywordOnLabelerError(t *testing.T) {
	items := []trss.Item{
		// Title that contains an AI keyword so keyword matcher → "ai".
		{ID: "hn-1", Title: "OpenAI releases GPT-5", Source: trss.ItemSource{Name: "Hacker News"}},
	}

	labeler := &stubLabeler{err: errors.New("boom")}

	clusters := ClusterWith(context.Background(), items, labeler)
	if len(clusters) != 1 {
		t.Fatalf("expected 1 cluster, got %d", len(clusters))
	}
	if clusters[0].Slug != "ai" {
		t.Errorf("keyword fallback should produce ai, got %q", clusters[0].Slug)
	}
}

func TestClusterWithNilLabelerEqualsCluster(t *testing.T) {
	items := []trss.Item{
		{ID: "hn-1", Title: "OpenAI releases GPT-5", Source: trss.ItemSource{Name: "Hacker News"}},
	}

	a := Cluster(items)
	b := ClusterWith(context.Background(), items, nil)

	if len(a) != len(b) {
		t.Fatalf("Cluster vs ClusterWith(nil) length mismatch: %d vs %d", len(a), len(b))
	}
	if a[0].Slug != b[0].Slug {
		t.Errorf("nil labeler should match Cluster(): %q vs %q", a[0].Slug, b[0].Slug)
	}
}

func TestClusterWithRepoOverrideBeatsLabeler(t *testing.T) {
	// Pure github-trending cluster should always be "repo", regardless
	// of what the labeler says — labels.go enforces this and we're
	// re-asserting it after the LLM upgrade pass.
	items := []trss.Item{
		{ID: "gh-1", URL: "https://github.com/foo/bar", Title: "foo/bar", Source: trss.ItemSource{Name: "GitHub Trending", Via: "github-trending"}},
	}

	labeler := &stubLabeler{
		slugForID: map[string]string{"gh-1": "devtools"}, // misclassification
	}

	clusters := ClusterWith(context.Background(), items, labeler)
	if len(clusters) != 1 {
		t.Fatalf("expected 1 cluster, got %d", len(clusters))
	}
	if clusters[0].Slug != "repo" {
		t.Errorf("repo override should win over labeler: got %q want repo", clusters[0].Slug)
	}
}
