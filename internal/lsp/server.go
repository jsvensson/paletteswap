package lsp

import (
	"sync"

	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
	"github.com/tliron/glsp/server"

	"github.com/tliron/commonlog"
	_ "github.com/tliron/commonlog/simple"
)

const serverName = "pstheme-lsp"

type Server struct {
	handler    protocol.Handler
	docs       *DocumentStore
	version    string
	mu         sync.RWMutex
	results    map[string]*AnalysisResult
	docVersion map[string]int // Track document versions to prevent stale diagnostics
}

func NewServer(version string) *Server {
	s := &Server{
		docs:       NewDocumentStore(),
		version:    version,
		results:    make(map[string]*AnalysisResult),
		docVersion: make(map[string]int),
	}

	s.handler = protocol.Handler{
		Initialize:                     s.initialize,
		Initialized:                    s.initialized,
		Shutdown:                       s.shutdown,
		SetTrace:                       s.setTrace,
		TextDocumentDidOpen:            s.textDocumentDidOpen,
		TextDocumentDidChange:          s.textDocumentDidChange,
		TextDocumentDidClose:           s.textDocumentDidClose,
		TextDocumentHover:              s.textDocumentHover,
		TextDocumentDefinition:         s.textDocumentDefinition,
		TextDocumentCompletion:         s.textDocumentCompletion,
		TextDocumentColor:              s.textDocumentDocumentColor,
		TextDocumentColorPresentation:  s.textDocumentColorPresentation,
		TextDocumentSemanticTokensFull: s.textDocumentSemanticTokensFull,
	}

	return s
}

func (s *Server) Run() error {
	commonlog.Configure(1, nil)
	srv := server.NewServer(&s.handler, serverName, false)
	return srv.RunStdio()
}

func (s *Server) initialize(_ *glsp.Context, params *protocol.InitializeParams) (any, error) {
	capabilities := s.handler.CreateServerCapabilities()

	syncKind := protocol.TextDocumentSyncKindFull
	capabilities.TextDocumentSync = &protocol.TextDocumentSyncOptions{
		OpenClose: &protocol.True,
		Change:    &syncKind,
	}
	capabilities.CompletionProvider = &protocol.CompletionOptions{
		TriggerCharacters: []string{"."},
	}
	capabilities.ColorProvider = true
	capabilities.SemanticTokensProvider = &protocol.SemanticTokensOptions{
		Legend: protocol.SemanticTokensLegend{
			TokenTypes:     semanticTokenTypes,
			TokenModifiers: semanticTokenModifiers,
		},
		Full: protocol.SemanticDelta{
			Delta: &protocol.False,
		},
	}

	return protocol.InitializeResult{
		Capabilities: capabilities,
		ServerInfo: &protocol.InitializeResultServerInfo{
			Name:    serverName,
			Version: &s.version,
		},
	}, nil
}

func (s *Server) initialized(_ *glsp.Context, _ *protocol.InitializedParams) error {
	return nil
}

func (s *Server) shutdown(_ *glsp.Context) error {
	protocol.SetTraceValue(protocol.TraceValueOff)
	return nil
}

func (s *Server) setTrace(_ *glsp.Context, params *protocol.SetTraceParams) error {
	protocol.SetTraceValue(params.Value)
	return nil
}

func (s *Server) textDocumentDidOpen(ctx *glsp.Context, params *protocol.DidOpenTextDocumentParams) error {
	uri := string(params.TextDocument.URI)
	s.docs.Open(uri, params.TextDocument.Text)
	s.mu.Lock()
	s.docVersion[uri] = 0
	s.mu.Unlock()
	s.analyzeAndPublish(ctx.Notify, uri, 0)
	return nil
}

func (s *Server) textDocumentDidChange(ctx *glsp.Context, params *protocol.DidChangeTextDocumentParams) error {
	uri := string(params.TextDocument.URI)

	// Increment document version for each change
	s.mu.Lock()
	s.docVersion[uri]++
	version := s.docVersion[uri]
	s.mu.Unlock()

	for _, change := range params.ContentChanges {
		switch c := change.(type) {
		case protocol.TextDocumentContentChangeEventWhole:
			s.docs.Update(uri, c.Text)
		case *protocol.TextDocumentContentChangeEvent:
			// Range-based changes should not occur with Full sync, but handle them just in case
			// by updating with the new text (treating it as a full document update)
			s.docs.Update(uri, c.Text)
		}
	}
	s.analyzeAndPublish(ctx.Notify, uri, version)
	return nil
}

func (s *Server) textDocumentDidClose(_ *glsp.Context, params *protocol.DidCloseTextDocumentParams) error {
	uri := string(params.TextDocument.URI)
	s.docs.Close(uri)
	s.mu.Lock()
	delete(s.results, uri)
	delete(s.docVersion, uri)
	s.mu.Unlock()
	return nil
}

func (s *Server) analyzeAndPublish(notify glsp.NotifyFunc, uri string, version int) {
	content, ok := s.docs.Get(uri)
	if !ok {
		return
	}

	result := Analyze(uri, content)

	s.mu.Lock()
	s.results[uri] = result
	currentVersion := s.docVersion[uri]
	s.mu.Unlock()

	// Only publish diagnostics if this is still the latest version
	// This prevents stale diagnostics from being published when rapid changes occur
	if version == currentVersion {
		go notify(protocol.ServerTextDocumentPublishDiagnostics, protocol.PublishDiagnosticsParams{
			URI:         protocol.DocumentUri(uri),
			Diagnostics: result.Diagnostics,
		})
	}
}

func (s *Server) getResult(uri string) *AnalysisResult {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.results[uri]
}

// textDocumentSemanticTokensFull handles textDocument/semanticTokens/full requests
func (s *Server) textDocumentSemanticTokensFull(_ *glsp.Context, params *protocol.SemanticTokensParams) (*protocol.SemanticTokens, error) {
	uri := string(params.TextDocument.URI)
	content, ok := s.docs.Get(uri)
	if !ok {
		return &protocol.SemanticTokens{Data: []uint32{}}, nil
	}

	data := semanticTokensFull(content)
	return &protocol.SemanticTokens{Data: data}, nil
}
