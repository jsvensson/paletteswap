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
	handler protocol.Handler
	docs    *DocumentStore
	version string
	mu      sync.RWMutex
	results map[string]*AnalysisResult
}

func NewServer(version string) *Server {
	s := &Server{
		docs:    NewDocumentStore(),
		version: version,
		results: make(map[string]*AnalysisResult),
	}

	s.handler = protocol.Handler{
		Initialize:                    s.initialize,
		Initialized:                   s.initialized,
		Shutdown:                      s.shutdown,
		SetTrace:                      s.setTrace,
		TextDocumentDidOpen:           s.textDocumentDidOpen,
		TextDocumentDidChange:         s.textDocumentDidChange,
		TextDocumentDidClose:          s.textDocumentDidClose,
		TextDocumentHover:             s.textDocumentHover,
		TextDocumentDefinition:        s.textDocumentDefinition,
		TextDocumentCompletion:        s.textDocumentCompletion,
		TextDocumentColor:             s.textDocumentDocumentColor,
		TextDocumentColorPresentation: s.textDocumentColorPresentation,
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
	s.analyzeAndPublish(ctx.Notify, uri)
	return nil
}

func (s *Server) textDocumentDidChange(ctx *glsp.Context, params *protocol.DidChangeTextDocumentParams) error {
	uri := string(params.TextDocument.URI)
	for _, change := range params.ContentChanges {
		if c, ok := change.(protocol.TextDocumentContentChangeEventWhole); ok {
			s.docs.Update(uri, c.Text)
		}
	}
	s.analyzeAndPublish(ctx.Notify, uri)
	return nil
}

func (s *Server) textDocumentDidClose(_ *glsp.Context, params *protocol.DidCloseTextDocumentParams) error {
	uri := string(params.TextDocument.URI)
	s.docs.Close(uri)
	s.mu.Lock()
	delete(s.results, uri)
	s.mu.Unlock()
	return nil
}

func (s *Server) analyzeAndPublish(notify glsp.NotifyFunc, uri string) {
	content, ok := s.docs.Get(uri)
	if !ok {
		return
	}

	result := Analyze(uri, content)

	s.mu.Lock()
	s.results[uri] = result
	s.mu.Unlock()

	go notify(protocol.ServerTextDocumentPublishDiagnostics, protocol.PublishDiagnosticsParams{
		URI:         protocol.DocumentUri(uri),
		Diagnostics: result.Diagnostics,
	})
}

func (s *Server) getResult(uri string) *AnalysisResult {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.results[uri]
}
