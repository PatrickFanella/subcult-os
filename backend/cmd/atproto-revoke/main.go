package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"git.subcult.tv/PatrickFanella/subcult-os/internal/app"
)

func main() {
	status := flag.Bool("status", false, "show aggregate queue counts without sending requests")
	limit := flag.Int("limit", 10, "maximum attempts in this batch (1-100)")
	watch := flag.Bool("watch", false, "process bounded batches every 30 seconds until stopped")
	flag.Parse()
	if flag.NArg() != 0 || *limit < 1 || *limit > 100 || (*watch && *status) {
		fmt.Fprintln(os.Stderr, "usage: atproto-revoke [-status | -watch] [-limit 1..100]")
		os.Exit(2)
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	config := app.LoadConfig()
	if config.DatabaseURL == "" {
		fmt.Fprintln(os.Stderr, "DATABASE_URL is required")
		os.Exit(1)
	}
	db, err := app.OpenDB(ctx, config.DatabaseURL)
	if err != nil {
		fmt.Fprintln(os.Stderr, "revocation database unavailable")
		os.Exit(1)
	}
	defer db.Close()
	for {
		batchCtx, cancel := context.WithTimeout(ctx, time.Duration(*limit)*45*time.Second+30*time.Second)
		result, err := app.RunATProtoRevocations(batchCtx, config, db, *limit, *status)
		cancel()
		if ctx.Err() != nil {
			return
		}
		if err != nil {
			fmt.Fprintln(os.Stderr, "revocation batch failed; inspect configuration and queue status")
			os.Exit(1)
		}
		if err := json.NewEncoder(os.Stdout).Encode(result); err != nil {
			os.Exit(1)
		}
		if !*watch {
			return
		}
		timer := time.NewTimer(30 * time.Second)
		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-timer.C:
		}
	}
}
