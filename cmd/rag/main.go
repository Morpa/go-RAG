package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/Morpa/go-rag/app"
	"github.com/Morpa/go-rag/config"
)

func main() {
	// we need to:
	// - set up the app
	// - set up configuration
	// - set up an the LLM client
	// - set up the Read-Eval-Print loop (REPL)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := app.Run(ctx, config.Load()); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
