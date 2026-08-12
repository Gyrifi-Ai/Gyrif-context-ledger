package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/gyrifi/gyrif-context-ledger/runtime/internal/bootstrap"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if err := bootstrap.Run(ctx, os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "gyrifi:", err)
		os.Exit(1)
	}
}
