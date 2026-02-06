package main

import (
	"github.com/tliron/commonlog"
	_ "github.com/tliron/commonlog/simple"
	"github.com/tliron/glsp/server"

	vibelsp "github.com/vibe-lang/vibe/pkg/lsp"
)

func main() {
	commonlog.Configure(1, nil)

	s := vibelsp.NewServer()
	srv := server.NewServer(&s.Handler, "vibe-lsp", false)
	_ = srv.RunStdio()
}
