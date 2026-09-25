package chunker

import (
	"strconv"
	"strings"
	"testing"
)

const fixture = `package demo

import "fmt"

// Greeter says hello.
type Greeter struct {
	Name string
}

const (
	A = 1
	// comment inside a group
	B = 2
)

var Default = Greeter{Name: "world"}

var (
	x int
	y string
)

func Hello(g Greeter) string {
	type inner struct{ v int }
	_ = inner{}
	return fmt.Sprintf("hi %s", g.Name)
}

func (g *Greeter) Rename(n string) {
	g.Name = n
}

func (s Set[T]) Add(v T) {}
`

func TestSymbolChunker_TopLevelDeclarations(t *testing.T) {
	chunks, err := NewSymbolChunker().Chunk("demo.go", []byte(fixture))
	if err != nil {
		t.Fatalf("Chunk failed: %v", err)
	}

	want := []struct {
		name, kind string
		start, end int
	}{
		{"Greeter", KindType, 6, 8},
		{"A+1", KindConst, 10, 14},
		{"Default", KindVar, 16, 16},
		{"x+1", KindVar, 18, 21},
		{"Hello", KindFunc, 23, 27},
		{"Greeter.Rename", KindMethod, 29, 31},
		{"Set.Add", KindMethod, 33, 33},
	}

	if len(chunks) != len(want) {
		for _, c := range chunks {
			t.Logf("got %s %s %d-%d", c.SymbolKind, c.SymbolName, c.StartLine, c.EndLine)
		}
		t.Fatalf("expected %d chunks, got %d", len(want), len(chunks))
	}

	for i, w := range want {
		c := chunks[i]
		if c.SymbolName != w.name || c.SymbolKind != w.kind || c.StartLine != w.start || c.EndLine != w.end {
			t.Errorf("chunk %d: got %s %q %d-%d, want %s %q %d-%d",
				i, c.SymbolKind, c.SymbolName, c.StartLine, c.EndLine, w.kind, w.name, w.start, w.end)
		}
		if c.Hash == "" {
			t.Errorf("chunk %d (%s): Hash not set", i, c.SymbolName)
		}
		if wantID := "demo.go:" + itoa(w.start) + "-" + itoa(w.end); c.ID != wantID {
			t.Errorf("chunk %d: ID = %q, want %q", i, c.ID, wantID)
		}
	}
}

func TestSymbolChunker_NestedTypeIsNotDuplicated(t *testing.T) {
	chunks, _ := NewSymbolChunker().Chunk("demo.go", []byte(fixture))
	for _, c := range chunks {
		if c.SymbolName == "inner" {
			t.Fatalf("type nested in a function body became its own chunk: %+v", c)
		}
	}
}

func TestSymbolChunker_OversizedFallsBackToWindows(t *testing.T) {
	var b strings.Builder
	b.WriteString("package big\n\nfunc Big() {\n")
	for i := 0; i < 120; i++ {
		b.WriteString("\t_ = 1\n")
	}
	b.WriteString("}\n")

	// Big spans lines 3..124.
	chunks, err := NewSymbolChunker(100).Chunk("big.go", []byte(b.String()))
	if err != nil {
		t.Fatalf("Chunk failed: %v", err)
	}
	if len(chunks) < 2 {
		t.Fatalf("expected oversized func to be split, got %d chunks", len(chunks))
	}

	first, last := chunks[0], chunks[len(chunks)-1]
	if first.StartLine != 3 {
		t.Errorf("first window should start at absolute line 3, got %d", first.StartLine)
	}
	if last.EndLine != 124 {
		t.Errorf("last window should end at absolute line 124, got %d", last.EndLine)
	}
	for _, c := range chunks {
		if c.SymbolName != "Big" || c.SymbolKind != KindFunc {
			t.Errorf("window %s lost its symbol: %q %q", c.ID, c.SymbolName, c.SymbolKind)
		}
		if !strings.HasPrefix(c.ID, "big.go:") || c.ID != "big.go:"+itoa(c.StartLine)+"-"+itoa(c.EndLine) {
			t.Errorf("inconsistent ID %q", c.ID)
		}
	}
}

func TestSymbolChunker_NonGoFallsBack(t *testing.T) {
	chunks, err := NewSymbolChunker().Chunk("README.md", []byte("# title\nbody\n"))
	if err != nil {
		t.Fatal(err)
	}
	if len(chunks) != 1 || chunks[0].SymbolKind != KindWindow {
		t.Fatalf("expected one window chunk, got %+v", chunks)
	}
}

func TestFixedWindowChunker(t *testing.T) {
	lines := func(n int) string {
		var b strings.Builder
		for i := 1; i <= n; i++ {
			b.WriteString("line\n")
		}
		return b.String()
	}

	tests := []struct {
		name            string
		window, overlap int
		content         string
		wantRanges      [][2]int
	}{
		{"empty", 5, 1, "", nil},
		{"only newline", 5, 1, "\n", nil},
		{"trailing newline is not a line", 5, 1, "a\nb\n", [][2]int{{1, 2}}},
		{"exact multiple of step", 4, 2, lines(6), [][2]int{{1, 4}, {3, 6}}},
		{"overlap >= window still terminates", 3, 3, lines(4), [][2]int{{1, 3}, {2, 4}}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			chunks, err := NewFixedWindowChunker(tt.window, tt.overlap).Chunk("f.txt", []byte(tt.content))
			if err != nil {
				t.Fatal(err)
			}
			if len(chunks) != len(tt.wantRanges) {
				t.Fatalf("got %d chunks, want %d: %+v", len(chunks), len(tt.wantRanges), chunks)
			}
			for i, r := range tt.wantRanges {
				if chunks[i].StartLine != r[0] || chunks[i].EndLine != r[1] {
					t.Errorf("chunk %d: got %d-%d, want %d-%d", i, chunks[i].StartLine, chunks[i].EndLine, r[0], r[1])
				}
				if chunks[i].SymbolKind != KindWindow || chunks[i].Hash == "" {
					t.Errorf("chunk %d: kind=%q hash=%q", i, chunks[i].SymbolKind, chunks[i].Hash)
				}
			}
		})
	}
}

func TestNewChunk_HashDependsOnContentOnly(t *testing.T) {
	a := newChunk("a.go", 1, 2, "same", "", KindWindow)
	b := newChunk("b.go", 5, 6, "same", "", KindWindow)
	c := newChunk("a.go", 1, 2, "different", "", KindWindow)
	if a.Hash != b.Hash {
		t.Error("identical content should hash identically regardless of location")
	}
	if a.Hash == c.Hash {
		t.Error("different content should hash differently")
	}
}

func itoa(n int) string { return strconv.Itoa(n) }
