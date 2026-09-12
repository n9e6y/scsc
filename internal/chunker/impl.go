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

func (c *FixedWindowChunker) Chunk(filePath string, content []byte) ([]Chunk, error) {
	lines := strings.Split(string(content), "\n")
	var chunks []Chunk

	if len(lines) == 0 || len(content) == 0 {
		return chunks, nil
	}

	step := c.windowSize - c.overlap
	if step <= 0 {
		step = 1 // Safety check
	}

	for i := 0; i < len(lines); i += step {
		end := i + c.windowSize
		if end > len(lines) {
			end = len(lines)
		}

		chunkContent := strings.Join(lines[i:end], "\n")
		startLine := i + 1
		endLine := end

		chunks = append(chunks, Chunk{
			ID:        fmt.Sprintf("%s:%d-%d", filePath, startLine, endLine),
			FilePath:  filePath,
			Content:   chunkContent,
			StartLine: startLine,
			EndLine:   endLine,
		})

		if end == len(lines) {
			break
		}
	}

	return chunks, nil
}

// SymbolChunker uses AST to extract functions and structs.
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

func (c *SymbolChunker) Chunk(filePath string, content []byte) ([]Chunk, error) {
	// If it's not a Go file, fall back immediately to fixed-window chunking
	if !strings.HasSuffix(filePath, ".go") {
		return c.fallback.Chunk(filePath, content)
	}

	parser := sitter.NewParser()
	defer parser.Close()

	parser.SetLanguage(c.language)
	tree := parser.Parse(content, nil)
	if tree == nil {
		return nil, fmt.Errorf("failed to parse file: %s", filePath)
	}
	defer tree.Close()

	var chunks []Chunk

	queryStr := `
		(function_declaration) @chunk
		(method_declaration) @chunk
		(type_declaration) @chunk
	`

	query, err := sitter.NewQuery(c.language, queryStr)
	if err != nil {
		return nil, err
	}
	defer query.Close()

	cursor := sitter.NewQueryCursor()
	defer cursor.Close()

	matches := cursor.Matches(query, tree.RootNode(), content)

	for match := matches.Next(); match != nil; match = matches.Next() {
		for _, capture := range match.Captures {
			node := capture.Node

			startLine := int(node.StartPosition().Row) + 1
			endLine := int(node.EndPosition().Row) + 1
			lineCount := endLine - startLine

			// If the function is larger than maxLines, fall back to sliding window for just this block
			if lineCount > c.maxLines {
				nodeContent := content[node.StartByte():node.EndByte()]
				subChunks, _ := c.fallback.Chunk(filePath, nodeContent)

				// Adjust the relative line numbers to absolute line numbers in the file
				for i := range subChunks {
					subChunks[i].StartLine += startLine - 1
					subChunks[i].EndLine += startLine - 1
					subChunks[i].ID = fmt.Sprintf("%s:%d-%d", filePath, subChunks[i].StartLine, subChunks[i].EndLine)
				}
				chunks = append(chunks, subChunks...)
				continue
			}

			chunk := Chunk{
				ID:        fmt.Sprintf("%s:%d", filePath, startLine),
				FilePath:  filePath,
				Content:   string(content[node.StartByte():node.EndByte()]),
				StartLine: startLine,
				EndLine:   endLine,
			}
			chunks = append(chunks, chunk)
		}
	}

	return chunks, nil
}
