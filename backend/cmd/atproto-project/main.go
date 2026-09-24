// Command atproto-project runs the allowlisted, restart-safe AT Protocol
// record projection (AT-01) and its backfill/rebuild/reconcile recovery
// tooling. See docs/development/projection.md.
//
// Default behavior prints aggregate, secret-free metrics (stored cursor,
// stream lag, row counts, quarantine count, records by status, and the last
// run per kind) without opening a network connection. Passing -run
// explicitly starts consuming the configured stream source; this requires
// AT_PROJECTION_ENABLED=true and AT_PROJECTION_SOURCE_URL to be set,
// mirroring how atproto-revoke and email-deliver gate their own opt-in side
// effects. -backfill, -rebuild and -reconcile are the operator entry points
// for the recovery tooling; -approve-authority/-approved-by/-note manage the
// approved-authority allowlist those commands require.
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
	atprotocol "git.subcult.tv/PatrickFanella/subcult-os/internal/atproto"
)

func main() {
	run := flag.Bool("run", false, "consume the configured stream source until stopped; requires AT_PROJECTION_ENABLED=true")
	backfillDID := flag.String("backfill", "", "list the three admitted collections from this approved authority DID's PDS and apply them")
	rebuild := flag.Bool("rebuild", false, "rebuild a shadow state from every approved authority and report the diff against stored records, without writing")
	reconcile := flag.Bool("reconcile", false, "rebuild and apply the shadow state for every approved authority")
	approveAuthorityDID := flag.String("approve-authority", "", "approve (or re-approve) a DID as a backfill authority; requires -approved-by")
	revokeAuthorityDID := flag.String("revoke-authority", "", "revoke a previously approved backfill authority DID")
	approvedBy := flag.String("approved-by", "", "person id recorded as approving -approve-authority")
	note := flag.String("note", "", "operator note recorded with -approve-authority")
	flag.Parse()
	exclusive := 0
	for _, set := range []bool{*run, *backfillDID != "", *rebuild, *reconcile, *approveAuthorityDID != "", *revokeAuthorityDID != ""} {
		if set {
			exclusive++
		}
	}
	if flag.NArg() != 0 || exclusive > 1 {
		fmt.Fprintln(os.Stderr, "usage: atproto-project [-run | -backfill did | -rebuild | -reconcile | -approve-authority did -approved-by person-id [-note text] | -revoke-authority did]")
		os.Exit(2)
	}
	if *approveAuthorityDID != "" && *approvedBy == "" {
		fmt.Fprintln(os.Stderr, "-approve-authority requires -approved-by")
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

	switch {
	case *approveAuthorityDID != "":
		if err := app.ApproveProjectionAuthority(ctx, db, *approveAuthorityDID, *approvedBy, *note); err != nil {
			fmt.Fprintln(os.Stderr, "approve authority failed")
			os.Exit(1)
		}
		return
	case *revokeAuthorityDID != "":
		if err := app.RevokeProjectionAuthority(ctx, db, *revokeAuthorityDID); err != nil {
			fmt.Fprintln(os.Stderr, "revoke authority failed")
			os.Exit(1)
		}
		return
	case *backfillDID != "":
		catalog, err := atprotocol.LoadEmbeddedLexiconCatalog()
		if err != nil {
			fmt.Fprintln(os.Stderr, "load admitted lexicon catalog failed")
			os.Exit(1)
		}
		result, err := app.RunProjectionBackfill(ctx, db, catalog, atprotocol.NewIdentityRecordLister(), *backfillDID)
		if err != nil {
			fmt.Fprintln(os.Stderr, "backfill failed; inspect the recorded run for detail")
		}
		if encErr := json.NewEncoder(os.Stdout).Encode(result); encErr != nil {
			os.Exit(1)
		}
		if err != nil {
			os.Exit(1)
		}
		return
	case *rebuild:
		diff, err := app.RunProjectionRebuild(ctx, db, atprotocol.NewIdentityRecordLister())
		if err != nil {
			fmt.Fprintln(os.Stderr, "rebuild failed")
			os.Exit(1)
		}
		if err := json.NewEncoder(os.Stdout).Encode(diff); err != nil {
			os.Exit(1)
		}
		return
	case *reconcile:
		catalog, err := atprotocol.LoadEmbeddedLexiconCatalog()
		if err != nil {
			fmt.Fprintln(os.Stderr, "load admitted lexicon catalog failed")
			os.Exit(1)
		}
		stats, diff, err := app.RunProjectionReconcile(ctx, db, catalog, atprotocol.NewIdentityRecordLister())
		if err != nil {
			fmt.Fprintln(os.Stderr, "reconcile failed")
			os.Exit(1)
		}
		if err := json.NewEncoder(os.Stdout).Encode(struct {
			Stats app.ProjectionStats `json:"stats"`
			Diff  app.ProjectionDiff  `json:"diff"`
		}{stats, diff}); err != nil {
			os.Exit(1)
		}
		return
	}

	if !*run {
		metrics, err := app.RunProjectionMetrics(ctx, db)
		if err != nil {
			fmt.Fprintln(os.Stderr, "projection status unavailable")
			os.Exit(1)
		}
		if err := json.NewEncoder(os.Stdout).Encode(metrics); err != nil {
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
