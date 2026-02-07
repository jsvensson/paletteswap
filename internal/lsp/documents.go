package lsp

import "sync"

// DocumentStore holds open document contents keyed by URI.
type DocumentStore struct {
	mu   sync.RWMutex
	docs map[string]string
}

func NewDocumentStore() *DocumentStore {
	return &DocumentStore{docs: make(map[string]string)}
}

func (s *DocumentStore) Open(uri, content string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.docs[uri] = content
}

func (s *DocumentStore) Update(uri, content string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.docs[uri] = content
}

func (s *DocumentStore) Close(uri string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.docs, uri)
}

func (s *DocumentStore) Get(uri string) (string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	content, ok := s.docs[uri]
	return content, ok
}
