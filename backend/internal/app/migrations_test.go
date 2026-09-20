package app

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"
	"testing"
	"testing/fstest"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestLoadMigrationsValidatesSequence(t *testing.T) {
	tests := []struct {
		name    string
		files   fstest.MapFS
		wantErr string
	}{
		{
			name: "ordered",
			files: fstest.MapFS{
				"schema.sql":                     &fstest.MapFile{Data: []byte(`select 1`)},
				"migrations/README.md":           &fstest.MapFile{Data: []byte("docs")},
				"migrations/000002_add_name.sql": &fstest.MapFile{Data: []byte(`select 2`)},
			},
		},
		{
			name: "gap",
			files: fstest.MapFS{
				"schema.sql":                     &fstest.MapFile{Data: []byte(`select 1`)},
				"migrations/000003_skip_two.sql": &fstest.MapFile{Data: []byte(`select 3`)},
			},
			wantErr: "got version 3, want 2",
		},
		{
			name: "invalid filename",
			files: fstest.MapFS{
				"schema.sql":                 &fstest.MapFile{Data: []byte(`select 1`)},
				"migrations/not_ordered.sql": &fstest.MapFile{Data: []byte(`select 2`)},
			},
			wantErr: "invalid migration filename",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			migrations, err := loadMigrations(tt.files)
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("loadMigrations() error = %v, want containing %q", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if len(migrations) != 2 || migrations[0].Version != 1 || migrations[1].Version != 2 {
				t.Fatalf("unexpected migrations: %#v", migrations)
			}
			if migrations[0].Checksum == "" || migrations[1].Checksum == "" {
				t.Fatal("expected migration checksums")
			}
		})
	}
}

func TestRunMigrationsTracksReplayAndVersion(t *testing.T) {
	pool := newMigrationTestPool(t)
	ctx := t.Context()

	if err := RunMigrations(ctx, pool); err != nil {
		t.Fatal(err)
	}
	if err := RunMigrations(ctx, pool); err != nil {
		t.Fatalf("replay migrations: %v", err)
	}

	version, err := CurrentSchemaVersion(ctx, pool)
	if err != nil {
		t.Fatal(err)
	}
	if version != minimumSchemaVersion {
		t.Fatalf("schema version = %d, want %d", version, minimumSchemaVersion)
	}
	var count int
	if err := pool.QueryRow(ctx, `select count(*) from schema_migrations`).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("migration ledger rows = %d, want 1", count)
	}
}

func TestRunMigrationsRejectsChangedAppliedMigration(t *testing.T) {
	pool := newMigrationTestPool(t)
	ctx := t.Context()
	if err := RunMigrations(ctx, pool); err != nil {
		t.Fatal(err)
	}

	changed := []migration{newMigration(1, "initial", []byte(`select 1`))}
	err := runMigrations(ctx, pool, changed)
	if err == nil || !strings.Contains(err.Error(), "changed after application") {
		t.Fatalf("runMigrations() error = %v, want changed migration rejection", err)
	}
}

func TestRunMigrationsRejectsDatabaseAheadOfBinary(t *testing.T) {
	pool := newMigrationTestPool(t)
	ctx := t.Context()
	if err := RunMigrations(ctx, pool); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `
		insert into schema_migrations (version, name, checksum)
		values (2, 'future', $1)
	`, strings.Repeat("0", 64)); err != nil {
		t.Fatal(err)
	}

	err := RunMigrations(ctx, pool)
	if err == nil || !strings.Contains(err.Error(), "does not match binary version") {
		t.Fatalf("RunMigrations() error = %v, want database-ahead rejection", err)
	}
}

func TestRunMigrationsRollsBackFailedVersion(t *testing.T) {
	pool := newMigrationTestPool(t)
	ctx := t.Context()
	migrations, err := loadMigrations(migrationFS)
	if err != nil {
		t.Fatal(err)
	}
	migrations = append(migrations, newMigration(2, "forced_failure", []byte(`
		create table migration_failure_probe (id integer primary key);
		select * from migration_table_that_does_not_exist;
	`)))

	err = runMigrations(ctx, pool, migrations)
	if err == nil || !strings.Contains(err.Error(), "apply migration 2 forced_failure") {
		t.Fatalf("runMigrations() error = %v, want forced migration failure", err)
	}

	var tableExists bool
	if err := pool.QueryRow(ctx, `select to_regclass('migration_failure_probe') is not null`).Scan(&tableExists); err != nil {
		t.Fatal(err)
	}
	if tableExists {
		t.Fatal("failed migration left its table behind")
	}
	var ledgerExists bool
	if err := pool.QueryRow(ctx, `select to_regclass('schema_migrations') is not null`).Scan(&ledgerExists); err != nil {
		t.Fatal(err)
	}
	if ledgerExists {
		t.Fatal("failed initial transaction left its migration ledger behind")
	}
}

func TestRunMigrationsSerializesConcurrentRunners(t *testing.T) {
	pool := newMigrationTestPool(t)
	ctx := t.Context()

	var wait sync.WaitGroup
	errorsFound := make(chan error, 2)
	for range 2 {
		wait.Add(1)
		go func() {
			defer wait.Done()
			errorsFound <- RunMigrations(ctx, pool)
		}()
	}
	wait.Wait()
	close(errorsFound)
	for err := range errorsFound {
		if err != nil {
			t.Fatal(err)
		}
	}

	var count int
	if err := pool.QueryRow(ctx, `select count(*) from schema_migrations`).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("migration ledger rows = %d, want 1", count)
	}
}

func newMigrationTestPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("set TEST_DATABASE_URL to run migration integration tests")
	}

	ctx := t.Context()
	basePool, err := OpenDB(ctx, databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	schema := fmt.Sprintf("migration_test_%d", time.Now().UnixNano())
	identifier := pgx.Identifier{schema}.Sanitize()
	if _, err := basePool.Exec(ctx, "create schema "+identifier); err != nil {
		basePool.Close()
		t.Fatal(err)
	}

	config, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		_, _ = basePool.Exec(context.Background(), "drop schema "+identifier+" cascade")
		basePool.Close()
		t.Fatal(err)
	}
	config.ConnConfig.RuntimeParams["search_path"] = schema
	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		_, _ = basePool.Exec(context.Background(), "drop schema "+identifier+" cascade")
		basePool.Close()
		t.Fatal(err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		_, _ = basePool.Exec(context.Background(), "drop schema "+identifier+" cascade")
		basePool.Close()
		t.Fatal(err)
	}

	t.Cleanup(func() {
		pool.Close()
		cleanupCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if _, err := basePool.Exec(cleanupCtx, "drop schema "+identifier+" cascade"); err != nil {
			t.Errorf("drop migration test schema: %v", err)
		}
		basePool.Close()
	})
	return pool
}
