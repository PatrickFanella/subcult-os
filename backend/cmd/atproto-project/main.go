// Command atproto-project runs the allowlisted, restart-safe AT Protocol
// record projection (AT-01). See docs/development/projection.md.
//
// Default behavior prints aggregate, secret-free status (stored cursor and
// row counts) without opening a network connection. Passing -run explicitly
// starts consuming the configured stream source; this requires
// AT_PROJECTION_ENABLED=true and AT_PROJECTION_SOURCE_URL to be set, mirroring
// how atproto-revoke and email-deliver gate their own opt-in side effects.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"git.subcult.tv/PatrickFanella/subcult-os/internal/app"
)

func main() {
	run := flag.Bool("run", false, "consume the configured stream source until stopped; requires AT_PROJECTION_ENABLED=true")
	flag.Parse()
	if flag.NArg() != 0 {
		fmt.Fprintln(os.Stderr, "usage: atproto-project [-run]")
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
		fmt.Fprintln(os.Stderr, "projection database unavailable")
		os.Exit(1)
	}
	defer db.Close()

	if !*run {
		status, err := app.RunProjectionStatus(ctx, db)
		if err != nil {
			fmt.Fprintln(os.Stderr, "projection status unavailable")
			os.Exit(1)
		}
		if err := json.NewEncoder(os.Stdout).Encode(status); err != nil {
			os.Exit(1)
		}
		return
	}

	if !config.ATProjectionEnabled {
		fmt.Fprintln(os.Stderr, "AT_PROJECTION_ENABLED must be true to run -run")
		os.Exit(1)
	}
	stats, err := app.RunProjection(ctx, config, db)
	if err != nil {
		fmt.Fprintln(os.Stderr, "projection run failed; inspect configuration and stored cursor")
		os.Exit(1)
	}
	if err := json.NewEncoder(os.Stdout).Encode(stats); err != nil {
		os.Exit(1)
	}
}
