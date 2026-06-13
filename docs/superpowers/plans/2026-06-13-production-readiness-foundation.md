# Production Readiness Foundation Implementation Plan

> **For agentic workers:** Execute this plan task-by-task. Recommended path:
> dispatch a fresh subagent per task, review each result with `review-quality`,
> then continue. For complex multi-agent splits, use
> `parallel-feature-development`, `team-composition-patterns`, and
> `team-communication-protocols`. Steps use checkbox (`- [ ]`) syntax for
> tracking.

**Goal:** Make the first lifecycle alpha safer to run outside a local-only environment before adding Stripe or larger product slices.

**Architecture:** Keep the monolithic Go API and Docker Compose setup. Add small, testable readiness gates around configuration, database availability, request observability, and deploy documentation without changing product behavior.

**Tech Stack:** Go 1.26, pgx/pgxpool, PostgreSQL 17, Docker Compose, Vite React TypeScript, Make.

---

## File Structure

- Modify `backend/internal/app/config.go`: add `Config.Validate()` with production-specific checks and typed error messages.
- Modify `backend/cmd/app/main.go`: validate config before opening the DB; fail fast in production when required settings are unsafe.
- Modify `backend/internal/app/app.go`: add `/api/ready` and keep `/api/health` as a cheap process health check.
- Modify `backend/internal/app/app_test.go`: add config validation tests and readiness endpoint tests without requiring a real DB.
- Modify `backend/internal/app/lifecycle_test.go`: add DB-backed readiness success coverage.
- Modify `.env.example`: document production-sensitive values without committing secrets.
- Modify `README.md`: add a short production-readiness checklist.
- Optional later: add `docs/runbooks/production-readiness.md` if README becomes too long.

---

### Task 1: Config Validation and Readiness Endpoint

**Files:**
- Modify: `backend/internal/app/config.go`
- Modify: `backend/cmd/app/main.go`
- Modify: `backend/internal/app/app.go`
- Modify: `backend/internal/app/app_test.go`
- Modify: `backend/internal/app/lifecycle_test.go`
- Modify: `.env.example`
- Modify: `README.md`

- [ ] **Step 1: Write config validation tests**

Add tests to `backend/internal/app/app_test.go`:

```go
func TestConfigValidateAllowsDevelopmentDefaults(t *testing.T) {
	config := Config{AppEnv: "development", Addr: ":8080"}
	if err := config.Validate(); err != nil {
		t.Fatalf("development config should validate: %v", err)
	}
}

func TestConfigValidateRejectsUnsafeProduction(t *testing.T) {
	config := Config{AppEnv: "production", Addr: ":8080", SessionSecret: "dev-session-secret-change-me"}
	err := config.Validate()
	if err == nil {
		t.Fatal("expected production config validation error")
	}
	message := err.Error()
	for _, want := range []string{"DATABASE_URL", "SESSION_SECRET", "PUBLIC_WEB_URL"} {
		if !strings.Contains(message, want) {
			t.Fatalf("expected %s in validation error, got %q", want, message)
		}
	}
}

func TestConfigValidateAllowsSafeProduction(t *testing.T) {
	config := Config{
		AppEnv:        "production",
		DatabaseURL:   "postgres://app:secret@db:5432/app?sslmode=require",
		SessionSecret: "replace-with-a-long-random-secret",
		PublicWebURL:  "https://subcult.example",
		Addr:          ":8080",
	}
	if err := config.Validate(); err != nil {
		t.Fatalf("production config should validate: %v", err)
	}
}
```

- [ ] **Step 2: Implement `Config.Validate()`**

Add to `backend/internal/app/config.go`:

```go
func (c Config) Validate() error {
	var problems []string
	if strings.TrimSpace(c.Addr) == "" {
		problems = append(problems, "API_ADDR is required")
	}
	if strings.EqualFold(strings.TrimSpace(c.AppEnv), "production") {
		if strings.TrimSpace(c.DatabaseURL) == "" {
			problems = append(problems, "DATABASE_URL is required in production")
		}
		if strings.TrimSpace(c.PublicWebURL) == "" {
			problems = append(problems, "PUBLIC_WEB_URL is required in production")
		}
		secret := strings.TrimSpace(c.SessionSecret)
		if secret == "" || secret == "dev-session-secret-change-me" || len(secret) < 24 {
			problems = append(problems, "SESSION_SECRET must be a non-default value with at least 24 characters in production")
		}
	}
	if len(problems) > 0 {
		return fmt.Errorf("invalid config: %s", strings.Join(problems, "; "))
	}
	return nil
}
```

- [ ] **Step 3: Wire config validation into startup**

In `backend/cmd/app/main.go`, after `config := app.LoadConfig()` add:

```go
if err := config.Validate(); err != nil {
	log.Fatal(err)
}
```

- [ ] **Step 4: Add readiness endpoint tests**

In `backend/internal/app/app_test.go`, add:

```go
func TestReadyWithoutDatabase(t *testing.T) {
	app := NewTestApp(t)
	req := httptest.NewRequest(http.MethodGet, "/api/ready", nil)
	rec := httptest.NewRecorder()
	app.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d: %s", rec.Code, rec.Body.String())
	}
}
```

In `backend/internal/app/lifecycle_test.go`, add:

```go
func TestReadyWithDatabase(t *testing.T) {
	fx := newLifecycleFixture(t)
	getJSON(t, fx.app, nil, "/api/ready", http.StatusOK)
}
```

- [ ] **Step 5: Implement `/api/ready`**

In `backend/internal/app/app.go`, register:

```go
a.mux.HandleFunc("GET /api/ready", a.handleReady)
```

Add:

```go
func (a *App) handleReady(w http.ResponseWriter, r *http.Request) {
	if a.db == nil {
		writeError(w, http.StatusServiceUnavailable, "database unavailable")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()
	if err := a.db.Ping(ctx); err != nil {
		writeError(w, http.StatusServiceUnavailable, "database unavailable")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ready"})
}
```

- [ ] **Step 6: Document config/readiness**

In `.env.example`, add comments that production must override `SESSION_SECRET`, `PUBLIC_WEB_URL`, and `DATABASE_URL`.

In `README.md`, add a “Production readiness checklist” section with:

```markdown
## Production readiness checklist

- Set `APP_ENV=production`.
- Set a real `DATABASE_URL`; production startup fails without it.
- Set a non-default `SESSION_SECRET` with at least 24 characters.
- Set `PUBLIC_WEB_URL` to the HTTPS web origin used by browsers.
- Use `/api/health` for process health and `/api/ready` for DB-backed readiness.
```

- [ ] **Step 7: Verify**

Run:

```bash
make verify
```

Expected: all checks pass.

Run DB-backed readiness tests when the local stack is running:

```bash
cd backend
set -a; source ../.env; set +a
TEST_DATABASE_URL="postgres://${POSTGRES_USER}:${POSTGRES_PASSWORD}@localhost:${POSTGRES_PORT}/${POSTGRES_DB}?sslmode=disable" go test ./internal/app
```

Expected: all backend app tests pass.

---

### Task 2: Request Logging Middleware

**Files:**
- Modify: `backend/internal/app/app.go`
- Test: `backend/internal/app/app_test.go`

- [ ] Add a small middleware around `originGuard` that logs method, path, status, and duration using the standard library logger.
- [ ] Add a response recorder type that captures status code while preserving `http.ResponseWriter` behavior.
- [ ] Add a test that `/api/health` still returns 200 through the middleware.
- [ ] Run `make verify`.

---

### Task 3: Migration/Schema Runbook

**Files:**
- Create: `docs/runbooks/database-migrations.md`
- Modify: `README.md`

- [ ] Document the current idempotent `schema.sql` migration approach.
- [ ] Document the current limitation: no ordered migration history yet.
- [ ] Document the trigger for adding a real migration tool before destructive schema changes or external production data.
- [ ] Link the runbook from README.
- [ ] Run `make verify`.

---

### Task 4: Deployment Checklist

**Files:**
- Create: `docs/runbooks/deployment-checklist.md`
- Modify: `README.md`

- [ ] Document build commands: `make verify`, `make build`, `make compose-config`.
- [ ] Document required env vars: `APP_ENV`, `DATABASE_URL`, `SESSION_SECRET`, `PUBLIC_WEB_URL`, `API_ADDR`.
- [ ] Document smoke checks: `/api/health`, `/api/ready`, app URL, `make alpha-qa` for local only.
- [ ] Document rollback assumption: keep previous image/artifact and database backup before migrations.
- [ ] Run `make verify`.

---

## Self-Review

- Spec coverage: config validation, readiness, logging, migration docs, and deploy docs are covered by Tasks 1-4.
- Placeholder scan: no `TBD`, vague “handle errors”, or missing command references remain.
- Type consistency: `Config.Validate`, `/api/ready`, and existing test helper names match current code.
