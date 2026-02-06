package lsp

import (
	"sync"

	"github.com/vibe-lang/vibe/pkg/ast"
)

// Document represents an open text document in the editor.
type Document struct {
	URI     string
	Content string
	Version int32
	AST     *ast.Program
	Errors  []string
}

// DocumentStore manages open documents in memory.
type DocumentStore struct {
	mu   sync.RWMutex
	docs map[string]*Document
}

// NewDocumentStore creates a new empty document store.
func NewDocumentStore() *DocumentStore {
	return &DocumentStore{
		docs: make(map[string]*Document),
	}
}

// Open stores a new document.
func (ds *DocumentStore) Open(uri string, content string, version int32) *Document {
	doc := &Document{
		URI:     uri,
		Content: content,
		Version: version,
	}
	ds.mu.Lock()
	ds.docs[uri] = doc
	ds.mu.Unlock()
	return doc
}

// Update replaces the content of a document.
func (ds *DocumentStore) Update(uri string, content string, version int32) *Document {
	ds.mu.Lock()
	defer ds.mu.Unlock()
	doc, ok := ds.docs[uri]
	if !ok {
		doc = &Document{URI: uri}
		ds.docs[uri] = doc
	}
	doc.Content = content
	doc.Version = version
	return doc
}

// Close removes a document from the store.
func (ds *DocumentStore) Close(uri string) {
	ds.mu.Lock()
	delete(ds.docs, uri)
	ds.mu.Unlock()
}

// Get retrieves a document by URI.
func (ds *DocumentStore) Get(uri string) (*Document, bool) {
	ds.mu.RLock()
	doc, ok := ds.docs[uri]
	ds.mu.RUnlock()
	return doc, ok
}
