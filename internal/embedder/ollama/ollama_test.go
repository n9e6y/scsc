package ollama

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// fakeOllama embeds each prompt as [len(prompt), 1, 2] so results can be traced back to inputs.
func fakeOllama(t *testing.T) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/embeddings" || r.Method != http.MethodPost {
			http.Error(w, "unexpected route", http.StatusNotFound)
			return
		}
		var req embedRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if req.Model != "test-model" {
			http.Error(w, "wrong model", http.StatusBadRequest)
			return
		}
		_ = json.NewEncoder(w).Encode(embedResponse{Embedding: []float32{float32(len(req.Prompt)), 1, 2}})
	}))
	t.Cleanup(srv.Close)
	return srv
}

func TestEmbed_PreservesOrderAndLearnsDims(t *testing.T) {
	o := New(fakeOllama(t).URL, "test-model")

	if o.Dimensions() != 0 {
		t.Fatalf("dims should be unknown before the first call, got %d", o.Dimensions())
	}

	texts := make([]string, 20) // more than maxConcurrentRequests, so requests overlap
	for i := range texts {
		texts[i] = strings.Repeat("x", i+1)
	}

	vecs, err := o.Embed(context.Background(), texts)
	if err != nil {
		t.Fatal(err)
	}
	for i, v := range vecs {
		if int(v[0]) != i+1 {
			t.Fatalf("vector %d belongs to input of length %d: order not preserved", i, int(v[0]))
		}
	}
	if o.Dimensions() != 3 {
		t.Errorf("Dimensions() = %d, want 3", o.Dimensions())
	}
	if o.Provider() != ProviderName || o.Model() != "test-model" {
		t.Errorf("unexpected provider/model %q/%q", o.Provider(), o.Model())
	}
}

func TestEmbed_ServerErrorIsReturned(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "model not found", http.StatusNotFound)
	}))
	t.Cleanup(srv.Close)

	if _, err := New(srv.URL, "missing").Embed(context.Background(), []string{"a", "b"}); err == nil {
		t.Fatal("expected an error for a non-200 response")
	}
}

func TestEmbed_EmptyEmbeddingIsAnError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"embedding": []}`))
	}))
	t.Cleanup(srv.Close)

	_, err := New(srv.URL, "llama3").Embed(context.Background(), []string{"a"})
	if err == nil || !strings.Contains(err.Error(), "empty embedding") {
		t.Fatalf("expected empty-embedding error, got %v", err)
	}
}
