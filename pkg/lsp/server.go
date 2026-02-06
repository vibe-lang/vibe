package lsp

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"

	"github.com/vibe-lang/vibe/pkg/ast"
	"github.com/vibe-lang/vibe/pkg/lexer"
	"github.com/vibe-lang/vibe/pkg/parser"
)

var serverName = "vibe-lsp"
var version = "0.1.0"

// Server is the Vibe LSP server.
type Server struct {
	Handler   protocol.Handler
	Documents *DocumentStore
}

// NewServer creates a new Vibe LSP server with all handlers registered.
func NewServer() *Server {
	s := &Server{
		Documents: NewDocumentStore(),
	}

	s.Handler = protocol.Handler{
		Initialize:                 s.initialize,
		Initialized:                s.initialized,
		Shutdown:                   s.shutdown,
		SetTrace:                   s.setTrace,
		TextDocumentDidOpen:        s.textDocumentDidOpen,
		TextDocumentDidChange:      s.textDocumentDidChange,
		TextDocumentDidClose:       s.textDocumentDidClose,
		TextDocumentDidSave:        s.textDocumentDidSave,
		TextDocumentCompletion:     s.textDocumentCompletion,
		TextDocumentHover:          s.textDocumentHover,
		TextDocumentDocumentSymbol: s.textDocumentDocumentSymbol,
		TextDocumentDefinition:     s.textDocumentDefinition,
	}

	return s
}

// ---------------------------------------------------------------------------
// Lifecycle
// ---------------------------------------------------------------------------

func (s *Server) initialize(context *glsp.Context, params *protocol.InitializeParams) (any, error) {
	capabilities := s.Handler.CreateServerCapabilities()

	// Override to full sync (simpler, fine for small files)
	syncKind := protocol.TextDocumentSyncKindFull
	capabilities.TextDocumentSync.(*protocol.TextDocumentSyncOptions).Change = &syncKind

	return protocol.InitializeResult{
		Capabilities: capabilities,
		ServerInfo: &protocol.InitializeResultServerInfo{
			Name:    serverName,
			Version: &version,
		},
	}, nil
}

func (s *Server) initialized(context *glsp.Context, params *protocol.InitializedParams) error {
	return nil
}

func (s *Server) shutdown(context *glsp.Context) error {
	return nil
}

func (s *Server) setTrace(context *glsp.Context, params *protocol.SetTraceParams) error {
	return nil
}

// ---------------------------------------------------------------------------
// Document Synchronization
// ---------------------------------------------------------------------------

func (s *Server) textDocumentDidOpen(context *glsp.Context, params *protocol.DidOpenTextDocumentParams) error {
	doc := s.Documents.Open(
		params.TextDocument.URI,
		params.TextDocument.Text,
		params.TextDocument.Version,
	)
	s.analyzeAndPublish(context, doc)
	return nil
}

func (s *Server) textDocumentDidChange(context *glsp.Context, params *protocol.DidChangeTextDocumentParams) error {
	for _, change := range params.ContentChanges {
		if c, ok := change.(protocol.TextDocumentContentChangeEventWhole); ok {
			doc := s.Documents.Update(
				params.TextDocument.URI,
				c.Text,
				params.TextDocument.Version,
			)
			s.analyzeAndPublish(context, doc)
		}
	}
	return nil
}

func (s *Server) textDocumentDidClose(context *glsp.Context, params *protocol.DidCloseTextDocumentParams) error {
	s.Documents.Close(params.TextDocument.URI)
	// Clear diagnostics
	context.Notify(protocol.ServerTextDocumentPublishDiagnostics, &protocol.PublishDiagnosticsParams{
		URI:         params.TextDocument.URI,
		Diagnostics: []protocol.Diagnostic{},
	})
	return nil
}

func (s *Server) textDocumentDidSave(context *glsp.Context, params *protocol.DidSaveTextDocumentParams) error {
	// Re-analyze on save (content may have been updated)
	if doc, ok := s.Documents.Get(params.TextDocument.URI); ok {
		s.analyzeAndPublish(context, doc)
	}
	return nil
}

// ---------------------------------------------------------------------------
// Diagnostics
// ---------------------------------------------------------------------------

// analyzeAndPublish parses the document and publishes diagnostics.
func (s *Server) analyzeAndPublish(context *glsp.Context, doc *Document) {
	l := lexer.New(doc.Content)
	p := parser.New(l)
	program := p.ParseProgram()

	doc.AST = program
	doc.Errors = p.Errors()

	diagnostics := make([]protocol.Diagnostic, 0, len(doc.Errors))

	// Parse error format: [line:col] message
	errorPattern := regexp.MustCompile(`\[(\d+):(\d+)\]\s*(.*)`)

	for _, errMsg := range doc.Errors {
		line := uint32(0)
		col := uint32(0)
		message := errMsg

		if matches := errorPattern.FindStringSubmatch(errMsg); matches != nil {
			if l, err := strconv.Atoi(matches[1]); err == nil && l > 0 {
				line = uint32(l - 1)
			}
			if c, err := strconv.Atoi(matches[2]); err == nil && c > 0 {
				col = uint32(c - 1)
			}
			message = matches[3]
		}

		severity := protocol.DiagnosticSeverityError
		diagnostics = append(diagnostics, protocol.Diagnostic{
			Range: protocol.Range{
				Start: protocol.Position{Line: line, Character: col},
				End:   protocol.Position{Line: line, Character: 1000},
			},
			Severity: &severity,
			Source:   &serverName,
			Message:  message,
		})
	}

	context.Notify(protocol.ServerTextDocumentPublishDiagnostics, &protocol.PublishDiagnosticsParams{
		URI:         doc.URI,
		Diagnostics: diagnostics,
	})
}

// ---------------------------------------------------------------------------
// Completion
// ---------------------------------------------------------------------------

func (s *Server) textDocumentCompletion(context *glsp.Context, params *protocol.CompletionParams) (any, error) {
	items := make([]protocol.CompletionItem, 0, len(keywords)+len(builtinNames)+len(typeNames))

	keywordKind := protocol.CompletionItemKindKeyword
	for _, kw := range keywords {
		items = append(items, protocol.CompletionItem{
			Label:  kw,
			Kind:   &keywordKind,
			Detail: stringPtr("keyword"),
		})
	}

	funcKind := protocol.CompletionItemKindFunction
	for name, doc := range builtinDocs {
		items = append(items, protocol.CompletionItem{
			Label:         name,
			Kind:          &funcKind,
			Detail:        stringPtr(doc.Signature),
			Documentation: doc.Description,
		})
	}

	typeKind := protocol.CompletionItemKindTypeParameter
	for _, t := range typeNames {
		items = append(items, protocol.CompletionItem{
			Label:  t,
			Kind:   &typeKind,
			Detail: stringPtr("type"),
		})
	}

	return items, nil
}

// ---------------------------------------------------------------------------
// Hover
// ---------------------------------------------------------------------------

func (s *Server) textDocumentHover(context *glsp.Context, params *protocol.HoverParams) (*protocol.Hover, error) {
	doc, ok := s.Documents.Get(params.TextDocument.URI)
	if !ok {
		return nil, nil
	}

	word := getWordAtPosition(doc.Content, params.Position)
	if word == "" {
		return nil, nil
	}

	if info, ok := builtinDocs[word]; ok {
		content := fmt.Sprintf("```vibe\n%s\n```", info.Signature)
		if info.MethodStyle != "" {
			content += fmt.Sprintf("\n\n*Also:* `%s`", info.MethodStyle)
		}
		content += "\n\n" + info.Description
		return &protocol.Hover{
			Contents: protocol.MarkupContent{
				Kind:  protocol.MarkupKindMarkdown,
				Value: content,
			},
		}, nil
	}

	// Check if it's a keyword with documentation
	if desc, ok := keywordDocs[word]; ok {
		return &protocol.Hover{
			Contents: protocol.MarkupContent{
				Kind:  protocol.MarkupKindMarkdown,
				Value: desc,
			},
		}, nil
	}

	return nil, nil
}

// ---------------------------------------------------------------------------
// Document Symbols (Outline)
// ---------------------------------------------------------------------------

func (s *Server) textDocumentDocumentSymbol(context *glsp.Context, params *protocol.DocumentSymbolParams) (any, error) {
	doc, ok := s.Documents.Get(params.TextDocument.URI)
	if !ok || doc.AST == nil {
		return nil, nil
	}

	symbols := []protocol.DocumentSymbol{}
	lines := strings.Split(doc.Content, "\n")

	defPattern := regexp.MustCompile(`^\s*def\s+([a-zA-Z_]\w*)`)
	classPattern := regexp.MustCompile(`^\s*class\s+([A-Z]\w*)`)
	structPattern := regexp.MustCompile(`^\s*struct\s+([A-Z]\w*)`)
	enumPattern := regexp.MustCompile(`^\s*enum\s+([A-Z]\w*)`)
	constPattern := regexp.MustCompile(`^\s*const\s+([A-Z_]\w*)\s*=`)

	type blockEntry struct {
		symbol *protocol.DocumentSymbol
		depth  int
	}
	stack := []blockEntry{}
	depth := 0

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		lineNum := uint32(i)

		var sym *protocol.DocumentSymbol
		var isBlock bool

		if matches := defPattern.FindStringSubmatch(line); matches != nil {
			kind := protocol.SymbolKindFunction
			sym = &protocol.DocumentSymbol{
				Name:           matches[1],
				Kind:           kind,
				Range:          lineRange(lineNum, uint32(len(line))),
				SelectionRange: lineRange(lineNum, uint32(len(line))),
			}
			isBlock = true
		} else if matches := classPattern.FindStringSubmatch(line); matches != nil {
			kind := protocol.SymbolKindClass
			sym = &protocol.DocumentSymbol{
				Name:           matches[1],
				Kind:           kind,
				Range:          lineRange(lineNum, uint32(len(line))),
				SelectionRange: lineRange(lineNum, uint32(len(line))),
			}
			isBlock = true
		} else if matches := structPattern.FindStringSubmatch(line); matches != nil {
			kind := protocol.SymbolKindStruct
			sym = &protocol.DocumentSymbol{
				Name:           matches[1],
				Kind:           kind,
				Range:          lineRange(lineNum, uint32(len(line))),
				SelectionRange: lineRange(lineNum, uint32(len(line))),
			}
			isBlock = true
		} else if matches := enumPattern.FindStringSubmatch(line); matches != nil {
			kind := protocol.SymbolKindEnum
			sym = &protocol.DocumentSymbol{
				Name:           matches[1],
				Kind:           kind,
				Range:          lineRange(lineNum, uint32(len(line))),
				SelectionRange: lineRange(lineNum, uint32(len(line))),
			}
			isBlock = true
		} else if matches := constPattern.FindStringSubmatch(line); matches != nil {
			kind := protocol.SymbolKindConstant
			sym = &protocol.DocumentSymbol{
				Name:           matches[1],
				Kind:           kind,
				Range:          lineRange(lineNum, uint32(len(line))),
				SelectionRange: lineRange(lineNum, uint32(len(line))),
			}
		}

		if sym != nil {
			// Add as child if inside a class/struct
			if len(stack) > 0 {
				parent := stack[len(stack)-1].symbol
				parent.Children = append(parent.Children, *sym)
			} else {
				symbols = append(symbols, *sym)
			}

			if isBlock {
				// Use pointer to the symbol we just appended
				var ptr *protocol.DocumentSymbol
				if len(stack) > 0 {
					parent := stack[len(stack)-1].symbol
					ptr = &parent.Children[len(parent.Children)-1]
				} else {
					ptr = &symbols[len(symbols)-1]
				}
				stack = append(stack, blockEntry{symbol: ptr, depth: depth})
			}
		}

		// Track depth
		if regexp.MustCompile(`^\s*(def|class|struct|enum|if|unless|until|while|for|case|try)\b`).MatchString(trimmed) {
			depth++
		}
		if strings.HasPrefix(trimmed, "end") && (len(trimmed) == 3 || trimmed[3] == ' ' || trimmed[3] == '\t' || trimmed[3] == '#') {
			depth--
			if len(stack) > 0 && depth <= stack[len(stack)-1].depth {
				top := stack[len(stack)-1]
				top.symbol.Range.End = protocol.Position{Line: lineNum, Character: uint32(len(line))}
				stack = stack[:len(stack)-1]
			}
		}
	}

	return symbols, nil
}

// ---------------------------------------------------------------------------
// Go to Definition
// ---------------------------------------------------------------------------

func (s *Server) textDocumentDefinition(context *glsp.Context, params *protocol.DefinitionParams) (any, error) {
	doc, ok := s.Documents.Get(params.TextDocument.URI)
	if !ok {
		return nil, nil
	}

	word := getWordAtPosition(doc.Content, params.Position)
	if word == "" {
		return nil, nil
	}

	lines := strings.Split(doc.Content, "\n")

	// Search for definition patterns
	patterns := []struct {
		re *regexp.Regexp
	}{
		{regexp.MustCompile(`^\s*def\s+` + regexp.QuoteMeta(word) + `\b`)},
		{regexp.MustCompile(`^\s*class\s+` + regexp.QuoteMeta(word) + `\b`)},
		{regexp.MustCompile(`^\s*struct\s+` + regexp.QuoteMeta(word) + `\b`)},
		{regexp.MustCompile(`^\s*enum\s+` + regexp.QuoteMeta(word) + `\b`)},
		{regexp.MustCompile(`^\s*const\s+` + regexp.QuoteMeta(word) + `\s*=`)},
		{regexp.MustCompile(`\b` + regexp.QuoteMeta(word) + `\s*=\s*`)},
	}

	for i, line := range lines {
		for _, pat := range patterns {
			if loc := pat.re.FindStringIndex(line); loc != nil {
				col := strings.Index(line[loc[0]:], word)
				if col >= 0 {
					col += loc[0]
				}
				return protocol.Location{
					URI: params.TextDocument.URI,
					Range: protocol.Range{
						Start: protocol.Position{Line: uint32(i), Character: uint32(col)},
						End:   protocol.Position{Line: uint32(i), Character: uint32(col + len(word))},
					},
				}, nil
			}
		}
	}

	return nil, nil
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func getWordAtPosition(content string, pos protocol.Position) string {
	lines := strings.Split(content, "\n")
	if int(pos.Line) >= len(lines) {
		return ""
	}
	line := lines[pos.Line]
	if int(pos.Character) >= len(line) {
		return ""
	}

	// Find word boundaries
	start := int(pos.Character)
	end := int(pos.Character)

	for start > 0 && isWordChar(line[start-1]) {
		start--
	}
	for end < len(line) && isWordChar(line[end]) {
		end++
	}

	if start == end {
		return ""
	}
	return line[start:end]
}

func isWordChar(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '_'
}

func lineRange(line uint32, endChar uint32) protocol.Range {
	return protocol.Range{
		Start: protocol.Position{Line: line, Character: 0},
		End:   protocol.Position{Line: line, Character: endChar},
	}
}

func stringPtr(s string) *string {
	return &s
}

// Suppress unused import warnings
var _ = ast.Node(nil)
var _ = lexer.New
var _ = parser.New
