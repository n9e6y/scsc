package chunker

import (
	"fmt"
	"strings"

	sitter "github.com/tree-sitter/go-tree-sitter"
	sitter_go "github.com/tree-sitter/tree-sitter-go/bindings/go"
)

// FixedWindowChunker breaks text into raw line blocks with overlap.
type FixedWindowChunker struct {
	windowSize int
	overlap    int
}

// NewFixedWindowChunker initializes a fallback chunker.
func NewFixedWindowChunker(windowSize, overlap int) *FixedWindowChunker {
	return &FixedWindowChunker{
		windowSize: windowSize,
		overlap:    overlap,
	}
}

func (c *FixedWindowChunker) Version() string { return WindowChunkerVersion }

func (c *FixedWindowChunker) Chunk(filePath string, content []byte) ([]Chunk, error) {
	return c.chunkAs(filePath, content, "", KindWindow), nil
}

// chunkAs windows content and labels every window with the given symbol, so the
// pieces of an oversized function still know which function they came from.
func (c *FixedWindowChunker) chunkAs(filePath string, content []byte, symbolName, symbolKind string) []Chunk {
	// A trailing newline would otherwise produce a phantom empty last line.
	text := strings.TrimSuffix(string(content), "\n")
	if text == "" {
		return nil
	}
	lines := strings.Split(text, "\n")

	step := c.windowSize - c.overlap
	if step <= 0 {
		step = 1 // Safety check
	}

	var chunks []Chunk
	for i := 0; i < len(lines); i += step {
		end := min(i+c.windowSize, len(lines))
		chunks = append(chunks, newChunk(filePath, i+1, end, strings.Join(lines[i:end], "\n"), symbolName, symbolKind))

		if end == len(lines) {
			break
		}
	}

	return chunks
}

// SymbolChunker uses the AST to extract top-level declarations
// (functions, methods, types, consts, vars).
type SymbolChunker struct {
	language *sitter.Language
	maxLines int
	fallback *FixedWindowChunker
}

// NewSymbolChunker accepts an optional maxLines argument (defaults to 100).
func NewSymbolChunker(maxLines ...int) *SymbolChunker {
	limit := 100
	if len(maxLines) > 0 {
		limit = maxLines[0]
	}
	return &SymbolChunker{
		language: sitter.NewLanguage(sitter_go.Language()),
		maxLines: limit,
		fallback: NewFixedWindowChunker(50, 10), // 50 lines, 10 line overlap
	}
}

func (c *SymbolChunker) Version() string { return SymbolChunkerVersion }

func (c *SymbolChunker) Chunk(filePath string, content []byte) ([]Chunk, error) {
	// If it's not a Go file, fall back immediately to fixed-window chunking
	if !strings.HasSuffix(filePath, ".go") {
		return c.fallback.Chunk(filePath, content)
	}

	// Parsers are cheap and not safe for concurrent use, so make one per call.
	parser := sitter.NewParser()
	defer parser.Close()

	if err := parser.SetLanguage(c.language); err != nil {
		return nil, fmt.Errorf("failed to set parser language: %w", err)
	}
	tree := parser.Parse(content, nil)
	if tree == nil {
		return nil, fmt.Errorf("failed to parse file: %s", filePath)
	}
	defer tree.Close()

	// Only walk the top level. A query would also match declarations nested inside
	// function bodies, duplicating code that is already part of the enclosing chunk.
	root := tree.RootNode()
	var chunks []Chunk

	for i := uint(0); i < root.NamedChildCount(); i++ {
		node := root.NamedChild(i)
		name, kind, ok := describeDeclaration(node, content)
		if !ok {
			continue // package clause, imports, comments
		}

		startLine := int(node.StartPosition().Row) + 1
		endLine := int(node.EndPosition().Row) + 1
		nodeContent := content[node.StartByte():node.EndByte()]

		// If the declaration is larger than maxLines, fall back to sliding window for just this block
		if endLine-startLine > c.maxLines {
			subChunks := c.fallback.chunkAs(filePath, nodeContent, name, kind)

			// Adjust the relative line numbers to absolute line numbers in the file
			for j := range subChunks {
				sc := subChunks[j]
				subChunks[j] = newChunk(filePath, sc.StartLine+startLine-1, sc.EndLine+startLine-1, sc.Content, name, kind)
			}
			chunks = append(chunks, subChunks...)
			continue
		}

		chunks = append(chunks, newChunk(filePath, startLine, endLine, string(nodeContent), name, kind))
	}

	return chunks, nil
}

// describeDeclaration returns the symbol name and kind for a top-level declaration node,
// or ok=false for nodes that should not become chunks.
func describeDeclaration(node *sitter.Node, src []byte) (name, kind string, ok bool) {
	switch node.Kind() {
	case "function_declaration":
		return fieldText(node, "name", src), KindFunc, true

	case "method_declaration":
		name := fieldText(node, "name", src)
		if recv := receiverTypeName(node, src); recv != "" {
			name = recv + "." + name
		}
		return name, KindMethod, true

	case "type_declaration":
		return groupName(specs(node, "type_spec", "type_alias"), src), KindType, true

	case "const_declaration":
		return groupName(specs(node, "const_spec"), src), KindConst, true

	case "var_declaration":
		// Parenthesized var blocks wrap their specs in a var_spec_list node.
		if list := firstNamedChildOfKind(node, "var_spec_list"); list != nil {
			node = list
		}
		return groupName(specs(node, "var_spec"), src), KindVar, true
	}
	return "", "", false
}

// receiverTypeName turns a receiver like "(s *Set[T])" into "Set".
func receiverTypeName(method *sitter.Node, src []byte) string {
	recv := method.ChildByFieldName("receiver")
	if recv == nil {
		return ""
	}
	param := firstNamedChildOfKind(recv, "parameter_declaration")
	if param == nil {
		return ""
	}
	typ := strings.TrimLeft(fieldText(param, "type", src), "*")
	if i := strings.IndexByte(typ, '['); i >= 0 {
		typ = typ[:i]
	}
	return typ
}

// specs returns the named children of node whose kind is one of kinds.
// Filtering by kind skips comments that sit inside grouped declarations.
func specs(node *sitter.Node, kinds ...string) []*sitter.Node {
	var out []*sitter.Node
	for i := uint(0); i < node.NamedChildCount(); i++ {
		child := node.NamedChild(i)
		for _, k := range kinds {
			if child.Kind() == k {
				out = append(out, child)
				break
			}
		}
	}
	return out
}

// groupName names a (possibly grouped) declaration after its first spec,
// e.g. "Version" or "KindFunc+5" for a const block with six specs.
func groupName(specs []*sitter.Node, src []byte) string {
	if len(specs) == 0 {
		return ""
	}
	name := fieldText(specs[0], "name", src)
	if len(specs) > 1 {
		name = fmt.Sprintf("%s+%d", name, len(specs)-1)
	}
	return name
}

func firstNamedChildOfKind(node *sitter.Node, kind string) *sitter.Node {
	for i := uint(0); i < node.NamedChildCount(); i++ {
		if child := node.NamedChild(i); child.Kind() == kind {
			return child
		}
	}
	return nil
}

func fieldText(node *sitter.Node, field string, src []byte) string {
	if child := node.ChildByFieldName(field); child != nil {
		return child.Utf8Text(src)
	}
	return ""
}
