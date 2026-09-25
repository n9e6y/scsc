package external

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

const keyEnv = "SEMCODE_TEST_EMBED_KEY"

func TestNew_MissingKey(t *testing.T) {
	t.Setenv(keyEnv, "")
	_, err := New("", keyEnv, "m")
	if err == nil || !strings.Contains(err.Error(), keyEnv) {
		t.Fatalf("expected an error naming %s, got %v", keyEnv, err)
	}
}

func TestEmbed_RequestShapeAndOutOfOrderResponse(t *testing.T) {
	t.Setenv(keyEnv, "secret-key")

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/embeddings" {
			http.Error(w, "unexpected route "+r.URL.Path, http.StatusNotFound)
			return
		}
		if got := r.Header.Get("Authorization"); got != "Bearer secret-key" {
			http.Error(w, "bad auth", http.StatusUnauthorized)
			return
		}
		var req embedRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Model != "m" || len(req.Input) != 2 {
			http.Error(w, "bad body", http.StatusBadRequest)
			return
		}
		// Deliberately return items out of order; the client must place them by index.
		_, _ = w.Write([]byte(`{"data":[
			{"index":1,"embedding":[2,2,2,2]},
			{"index":0,"embedding":[1,1,1,1]}
		]}`))
	}))
	t.Cleanup(srv.Close)

	e, err := New(srv.URL, keyEnv, "m")
	if err != nil {
		t.Fatal(err)
	}
	vecs, err := e.Embed(context.Background(), []string{"first", "second"})
	if err != nil {
		t.Fatal(err)
	}
	if vecs[0][0] != 1 || vecs[1][0] != 2 {
		t.Errorf("vectors not placed by index: %v", vecs)
	}
	if e.Dimensions() != 4 {
		t.Errorf("Dimensions() = %d, want 4", e.Dimensions())
	}
}

func TestEmbed_ErrorsDoNotLeakKey(t *testing.T) {
	t.Setenv(keyEnv, "secret-key")

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "rate limited", http.StatusTooManyRequests)
	}))
	t.Cleanup(srv.Close)

	e, _ := New(srv.URL, keyEnv, "m")
	_, err := e.Embed(context.Background(), []string{"a"})
	if err == nil {
		t.Fatal("expected an error for a non-200 response")
	}
	if strings.Contains(err.Error(), "secret-key") {
		t.Fatalf("error leaks the API key: %v", err)
	}
}

func TestEmbed_MissingItemIsAnError(t *testing.T) {
	t.Setenv(keyEnv, "k")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"data":[{"index":0,"embedding":[1]}]}`))
	}))
	t.Cleanup(srv.Close)

	e, _ := New(srv.URL, keyEnv, "m")
	if _, err := e.Embed(context.Background(), []string{"a", "b"}); err == nil {
		t.Fatal("expected an error when the API returns fewer embeddings than inputs")
	}
}
