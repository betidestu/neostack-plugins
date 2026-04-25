package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	editor, err := discover()
	if err != nil {
		var de *discoveryError
		if errors.As(err, &de) {
			fmt.Fprintf(os.Stderr, "\nneostack-connect: %s\n", de.msg)
			if de.hint != "" {
				fmt.Fprintf(os.Stderr, "hint: %s\n\n", de.hint)
			}
			os.Exit(1)
		}
		fmt.Fprintf(os.Stderr, "neostack-connect: fatal: %v\n", err)
		os.Exit(2)
	}

	fmt.Fprintf(os.Stderr,
		"neostack-connect: connecting to '%s' at %s\n",
		editor.ProjectName, editor.URL)

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
		fmt.Fprintf(os.Stderr, "neostack-connect: %v\n", err)
		os.Exit(2)
	}
}
