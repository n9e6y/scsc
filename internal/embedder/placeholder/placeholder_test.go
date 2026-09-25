package placeholder

import (
	"context"
	"math"
	"slices"
	"testing"
)

func dot(a, b []float32) float32 {
	var s float32
	for i := range a {
		s += a[i] * b[i]
	}
	return s
}

func TestEmbed_DeterministicAndNormalized(t *testing.T) {
	p := New()
	texts := []string{"func Login(user string) error", "func Login(user string) error", ""}

	vecs, err := p.Embed(context.Background(), texts)
	if err != nil {
		t.Fatal(err)
	}
	if len(vecs) != len(texts) {
		t.Fatalf("got %d vectors for %d texts", len(vecs), len(texts))
	}
	for i, v := range vecs {
		if len(v) != p.Dimensions() {
			t.Fatalf("vector %d has %d dims, want %d", i, len(v), p.Dimensions())
		}
	}

	if !slices.Equal(vecs[0], vecs[1]) {
		t.Error("identical text should produce identical vectors")
	}
	if norm := math.Sqrt(float64(dot(vecs[0], vecs[0]))); math.Abs(norm-1) > 1e-5 {
		t.Errorf("vector should be L2-normalized, norm = %f", norm)
	}
	if dot(vecs[2], vecs[2]) != 0 {
		t.Error("empty text should produce a zero vector")
	}
}

func TestEmbed_SimilarTextScoresHigher(t *testing.T) {
	vecs, err := New().Embed(context.Background(), []string{
		"open database connection pool",
		"close database connection pool",
		"render html template for the homepage",
	})
	if err != nil {
		t.Fatal(err)
	}
	if similar, unrelated := dot(vecs[0], vecs[1]), dot(vecs[0], vecs[2]); similar <= unrelated {
		t.Errorf("similar=%f should exceed unrelated=%f", similar, unrelated)
	}
}
