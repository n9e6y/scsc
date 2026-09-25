package embedder

import "testing"

func TestNew_KnownProviders(t *testing.T) {
	t.Setenv("SEMCODE_TEST_KEY", "k")
	opts := Options{OllamaModel: "nomic-embed-text", ExternalModel: "m", ExternalKeyEnv: "SEMCODE_TEST_KEY"}

	for _, p := range []string{ProviderPlaceholder, ProviderOllama, ProviderExternal} {
		e, err := New(p, opts)
		if err != nil {
			t.Fatalf("New(%q): %v", p, err)
		}
		if e.Provider() != p {
			t.Errorf("New(%q).Provider() = %q", p, e.Provider())
		}
	}
}

func TestNew_UnknownProviderIsAnError(t *testing.T) {
	if _, err := New("openia", Options{}); err == nil {
		t.Fatal("expected an error for an unknown provider instead of a silent fallback")
	}
}
