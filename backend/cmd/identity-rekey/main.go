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

// identity-rekey re-encrypts every IDENTITY_PROTECTION_KEY-protected row
// (verified emails, AT OAuth session and revocation payloads) from
// IDENTITY_PROTECTION_KEY_PREVIOUS to IDENTITY_PROTECTION_KEY. It is an
// explicit operator command, independent of whether the API is serving
// traffic, and prints aggregate counts only -- never an email, ciphertext,
// DID or session identifier.
func main() {
	status := flag.Bool("status", false, "show aggregate row counts without decrypting or rewriting anything")
	limit := flag.Int("limit", 500, "maximum rows per table in this batch (1-1000)")
	watch := flag.Bool("watch", false, "process bounded batches every 10 seconds until stopped")
	flag.Parse()
	if flag.NArg() != 0 || *limit < 1 || *limit > 1000 || (*watch && *status) {
		fmt.Fprintln(os.Stderr, "usage: identity-rekey [-status | -watch] [-limit 1..1000]")
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
		fmt.Fprintln(os.Stderr, "identity rekey database unavailable")
		os.Exit(1)
	}
	defer db.Close()
	for {
		batchCtx, cancel := context.WithTimeout(ctx, 2*time.Minute)
		result, err := app.RunIdentityRekey(batchCtx, config, db, *limit, *status)
		cancel()
		if ctx.Err() != nil {
			return
		}
		if err != nil {
			fmt.Fprintln(os.Stderr, "identity rekey batch failed; inspect configuration and row counts")
			os.Exit(1)
		}
		if err := json.NewEncoder(os.Stdout).Encode(result); err != nil {
			os.Exit(1)
		}
		if !*watch {
			return
		}
		timer := time.NewTimer(10 * time.Second)
		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-timer.C:
		}
	}
}
