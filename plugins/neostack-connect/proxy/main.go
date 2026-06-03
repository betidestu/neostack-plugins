package main

import (
	"context"
	"errors"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	editor, err := discover()
	if err != nil {
		var de *discoveryError
		if errors.As(err, &de) {
			// Don't exit — serve a degraded MCP server so the LLM sees the
			// problem via the unreal_status tool and can relay the fix.
			runDiagnosticMode(de)
			return
		}
		os.Exit(2)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigs
		cancel()
	}()

	bridge := newHTTPBridge(editor.URL)
	if err := bridge.run(ctx); err != nil {
		os.Exit(2)
	}
}
