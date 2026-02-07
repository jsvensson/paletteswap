package lsp

import (
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
}

func NewServer(version string) *Server {
	s := &Server{
		docs:    NewDocumentStore(),
		version: version,
	}

	s.handler = protocol.Handler{
		Initialize:            s.initialize,
		Initialized:           s.initialized,
		Shutdown:              s.shutdown,
		SetTrace:              s.setTrace,
		TextDocumentDidOpen:   s.textDocumentDidOpen,
		TextDocumentDidChange: s.textDocumentDidChange,
		TextDocumentDidClose:  s.textDocumentDidClose,
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

func (s *Server) textDocumentDidOpen(_ *glsp.Context, params *protocol.DidOpenTextDocumentParams) error {
	s.docs.Open(string(params.TextDocument.URI), params.TextDocument.Text)
	return nil
}

func (s *Server) textDocumentDidChange(_ *glsp.Context, params *protocol.DidChangeTextDocumentParams) error {
	for _, change := range params.ContentChanges {
		if c, ok := change.(protocol.TextDocumentContentChangeEventWhole); ok {
			s.docs.Update(string(params.TextDocument.URI), c.Text)
		}
	}
	return nil
}

func (s *Server) textDocumentDidClose(_ *glsp.Context, params *protocol.DidCloseTextDocumentParams) error {
	s.docs.Close(string(params.TextDocument.URI))
	return nil
}
