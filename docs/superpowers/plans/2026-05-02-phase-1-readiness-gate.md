# Phase 1 Readiness Gate Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make the scaffold's documented bootstrap and final readiness gate pass without hidden local state.

**Architecture:** Keep the existing Go, Makefile, Docker Compose, and test structure. The main change is to make `bootstrap-smoke` the isolated proof path and have `ready-for-adopters` depend on that proof instead of directly running integration tests against whatever happens to be on localhost. Local `.env` loading is added at the Makefile boundary so the README quickstart works as written.

**Tech Stack:** Go 1.25, Make, Docker Compose, goose, golangci-lint, chi, pgx/sqlc, Redis, Postgres

---

## File Map

- Modify: `Makefile`
  Loads `.env` when present, exports it to child commands, and changes `ready-for-adopters` to use the isolated bootstrap gate.
- Modify: `scripts/ci/bootstrap-smoke.sh`
  Runs unit and integration tests with race detection inside the isolated smoke environment.
- Modify: `tests/integration/testhelpers_test.go`
  Aligns fallback Postgres credentials with `.env.example` and `docker-compose.yml`.
- Create: `.dockerignore`
  Excludes local secrets, Git metadata, worktrees, build output, and test artifacts from Docker build context.
- Modify: `scripts/seed.sh`
  Uses a valid password and reports API failures accurately.
- Modify: `README.md`
  Replaces overclaims and documents the corrected quickstart and readiness gate.
- Modify: `docs/startup-readiness.md`
  Aligns the gate definition with the new self-contained readiness target.
- Modify: `docs/adoption-checklist.md`
  Keeps production caveats explicit and references the corrected readiness gate.
- Test: existing unit, integration, bootstrap, and Docker build commands.

### Task 1: Make The Final Readiness Gate Self-Contained

**Files:**
- Modify: `Makefile`
- Modify: `scripts/ci/bootstrap-smoke.sh`
- Modify: `docs/startup-readiness.md`
- Modify: `README.md`

- [ ] **Step 1: Reproduce the current broken gate**

Run:

```bash
make ready-for-adopters
```

Expected before the fix: FAIL during `make test-ci` integration tests if the local database is not already migrated with the expected credentials. The failure looks like:

```text
failed to ping test database
password authentication failed for user "user"
make[1]: *** [Makefile:30: test-ci] Error 1
```

- [ ] **Step 2: Update `bootstrap-smoke` to run race-enabled tests**

In `scripts/ci/bootstrap-smoke.sh`, replace the final two `go test` commands:

```bash
go test ./...
go test ./tests/integration/... -tags=integration
```

with:

```bash
go test ./... -race
go test ./tests/integration/... -v -race -tags=integration
```

This makes the isolated bootstrap proof cover the same race-enabled test strength currently expected from `test-ci`.

- [ ] **Step 3: Change `ready-for-adopters` to use the isolated proof path**

In `Makefile`, replace:

```make
ready-for-adopters:
	$(MAKE) lint
	$(MAKE) test-ci
	$(MAKE) bootstrap-smoke
	docker build -t go-backend-scaffold:ready .
```

with:

```make
ready-for-adopters:
	$(MAKE) lint
	$(MAKE) bootstrap-smoke
	docker build -t go-backend-scaffold:ready .
```

Keep `test-ci` unchanged for CI environments that have already provisioned and migrated Postgres and Redis:

```make
test-ci:
	go test ./... -race
	go test ./tests/integration/... -v -race -tags=integration
```

- [ ] **Step 4: Update `docs/startup-readiness.md` gate definition**

Replace the list of checks under "What The Gate Proves" with:

```markdown
`make ready-for-adopters` is the final local release gate for this repository. It runs:

1. `golangci-lint run ./...`
2. `make bootstrap-smoke`
3. `docker build -t go-backend-scaffold:ready .`

`bootstrap-smoke` creates isolated Postgres and Redis services, applies migrations, and runs the unit and integration test suites with race detection.
```

Also keep a short caveat:

```markdown
`make test-ci` remains available for CI jobs that have already provisioned Postgres and Redis. It is not the clean-checkout proof path by itself.
```

- [ ] **Step 5: Update the README quality gate paragraph**

In `README.md`, replace the paragraph that says the final local gate runs `make lint`, `make test-ci`, `make bootstrap-smoke`, and Docker build with:

```markdown
`make ready-for-adopters` is the final local release gate for this scaffold. It runs lint, the isolated bootstrap smoke path, and a Docker build. The bootstrap smoke path starts temporary Postgres and Redis services, applies migrations, and runs unit plus integration tests with race detection.
```

- [ ] **Step 6: Verify the targeted bootstrap path**

Run:

```bash
make bootstrap-smoke
```

Expected after the fix: PASS. The output should include migration success and race-enabled test passes:

```text
goose: successfully migrated database to version: 4
ok  	github.com/nyashahama/go-backend-scaffold/tests/integration
```

- [ ] **Step 7: Commit**

```bash
git add Makefile scripts/ci/bootstrap-smoke.sh README.md docs/startup-readiness.md
git commit -m "fix: make readiness gate self-contained"
```

### Task 2: Make Local `.env` And Integration Defaults Match The Quickstart

**Files:**
- Modify: `Makefile`
- Modify: `tests/integration/testhelpers_test.go`
- Modify: `README.md`

- [ ] **Step 1: Confirm the Makefile currently does not load `.env`**

Run:

```bash
cp .env.example .env
make migrate-status
```

Expected before the fix: FAIL or use an empty `DATABASE_URL` unless the shell already exported it. The failure may look like:

```text
goose -dir db/migrations postgres "" status
```

If you created `.env` solely for this check and it does not contain local secrets you need to keep, remove it:

```bash
rm .env
```

- [ ] **Step 2: Add optional `.env` loading to the top of `Makefile`**

Add this block after the `.PHONY` declaration and before the section comments:

```make
ifneq (,$(wildcard .env))
include .env
export
endif
```

The `include` line reads simple `KEY=value` entries from `.env`. The bare `export` exports Make variables to recipes, which lets `goose`, `go run`, and Docker-related commands see `DATABASE_URL`, `REDIS_URL`, and `JWT_SECRET`.

- [ ] **Step 3: Align the integration fallback Postgres URL**

In `tests/integration/testhelpers_test.go`, replace:

```go
dbURL = "postgres://user:password@localhost:5432/scaffold?sslmode=disable"
```

with:

```go
dbURL = "postgres://user:change-me-local-dev@localhost:5432/scaffold?sslmode=disable"
```

Do not change the Redis fallback:

```go
redisURL = "redis://localhost:6379"
```

- [ ] **Step 4: Update README quickstart language around `.env`**

In the "Verified Local Bootstrap" section, use this command block:

````markdown
```bash
cp .env.example .env
# Edit .env and replace JWT_SECRET before running the server.
make docker-up
make migrate-up
make test-ci
make run
```
````

Add one sentence after the block:

```markdown
The Makefile loads `.env` automatically when it exists, so the migration and run targets use the values from that file.
```

- [ ] **Step 5: Verify migration target receives `.env`**

Run:

```bash
cp .env.example .env
sed -i 's/JWT_SECRET=.*/JWT_SECRET=local-development-secret-that-is-at-least-32-chars/' .env
make docker-up
make migrate-up
go test ./tests/integration/... -v -race -tags=integration
make docker-down
```

Expected after the fix: migrations apply and integration tests pass using values loaded from `.env`.

- [ ] **Step 6: Commit**

```bash
git add Makefile tests/integration/testhelpers_test.go README.md
git commit -m "fix: align local environment bootstrap"
```

### Task 3: Add Docker Build Context Hygiene

**Files:**
- Create: `.dockerignore`
- Modify: `README.md`

- [ ] **Step 1: Confirm `.dockerignore` is missing**

Run:

```bash
test -f .dockerignore
```

Expected before the fix: command exits non-zero.

- [ ] **Step 2: Create `.dockerignore`**

Create `.dockerignore` with:

```dockerignore
.git
.gitignore
.worktrees
.claude

.env
.env.*
!.env.example

bin/
coverage.out
coverage.html
*.test
*.out

.DS_Store
Thumbs.db
.idea/
.vscode/
*.swp
*.swo
```

This keeps local secrets and irrelevant metadata out of the Docker context while preserving `.env.example` for documentation.

- [ ] **Step 3: Build the image**

Run:

```bash
docker build -t go-backend-scaffold:dockerignore-check .
```

Expected after the fix: PASS.

- [ ] **Step 4: Document the expectation**

In the README quality gate section, add:

```markdown
The Docker build uses `.dockerignore` to keep local secrets, Git metadata, and temporary worktrees out of the build context.
```

- [ ] **Step 5: Commit**

```bash
git add .dockerignore README.md
git commit -m "chore: add docker build context hygiene"
```

### Task 4: Fix The Seed Script Example

**Files:**
- Modify: `scripts/seed.sh`

- [ ] **Step 1: Confirm the current seed password is invalid**

Read the current payload:

```bash
sed -n '1,80p' scripts/seed.sh
```

Expected before the fix: the password is `dev-password-123`, which fails the auth password policy because it has no uppercase letter.

- [ ] **Step 2: Replace the seed script API call**

Replace the `curl | jq || echo` block in `scripts/seed.sh` with:

```bash
payload='{"email":"admin@example.com","password":"DevPassword123!","full_name":"Dev Admin"}'

response="$(curl -sS -X POST "$BASE_URL/api/v1/auth/register" \
  -H "Content-Type: application/json" \
  -d "$payload" \
  -w '\n%{http_code}')"

status="${response##*$'\n'}"
body="${response%$'\n'*}"

case "$status" in
  201)
    if command -v jq >/dev/null 2>&1; then
      printf '%s\n' "$body" | jq .
    else
      printf '%s\n' "$body"
    fi
    ;;
  409)
    echo "Seed user already exists."
    ;;
  *)
    echo "Seed failed with HTTP $status" >&2
    printf '%s\n' "$body" >&2
    exit 1
    ;;
esac
```

Keep the existing `BASE_URL` default:

```bash
BASE_URL="${BASE_URL:-http://localhost:8080}"
```

- [ ] **Step 3: Syntax-check the script**

Run:

```bash
bash -n scripts/seed.sh
```

Expected after the fix: PASS with no output.

- [ ] **Step 4: Commit**

```bash
git add scripts/seed.sh
git commit -m "fix: repair seed script example"
```

### Task 5: Correct Adopter-Facing Claims

**Files:**
- Modify: `README.md`
- Modify: `docs/startup-readiness.md`
- Modify: `docs/adoption-checklist.md`

- [ ] **Step 1: Replace the README opening claim**

In `README.md`, replace:

```markdown
A production-ready Go backend scaffold. Clone it, initialize the module path safely, and build your next API.
```

with:

```markdown
A production-oriented Go backend scaffold for startup APIs. Clone it, initialize the module path safely, verify the local bootstrap, and build your next API.
```

- [ ] **Step 2: Add a short non-drop-in caveat near the quickstart**

After the stack line or quickstart intro, add:

```markdown
This is a strong starting point, not a production drop-in. Before a real deploy, complete the adoption checklist and add product-specific authorization, deployment, backup, monitoring, and compliance decisions.
```

- [ ] **Step 3: Update `docs/startup-readiness.md` with the corrected gate language**

Ensure the document says:

````markdown
This scaffold is "ready to hand to a random startup for evaluation" only when maintainers can run:

```bash
make ready-for-adopters
```

and the command completes successfully on the current branch.
````

Keep the "What The Gate Does Not Prove" section and make sure it still lists:

```markdown
- the adoption checklist
- startup-specific production hardening
- environment-specific deployment validation
- legal, privacy, security review, or compliance work
```

- [ ] **Step 4: Update adoption checklist wording**

In `docs/adoption-checklist.md`, add this bullet to the "Repository Initialization" section:

```markdown
- Run `make ready-for-adopters` before using the initialized scaffold as your project baseline.
```

Do not remove the existing caveat:

```markdown
Use this scaffold as a starting point, not a production-ready drop-in.
```

- [ ] **Step 5: Commit**

```bash
git add README.md docs/startup-readiness.md docs/adoption-checklist.md
git commit -m "docs: clarify scaffold readiness claims"
```

### Task 6: Run Final Verification

**Files:**
- No source changes unless a verification failure reveals a real defect.

- [ ] **Step 1: Check formatting impact**

Run:

```bash
gofmt -w tests/integration/testhelpers_test.go
```

Expected: no functional change beyond normal Go formatting.

- [ ] **Step 2: Run unit/race sweep**

Run:

```bash
go test ./... -race
```

Expected: PASS.

- [ ] **Step 3: Run isolated bootstrap smoke**

Run:

```bash
make bootstrap-smoke
```

Expected: PASS, with temporary Docker Compose services cleaned up afterward.

- [ ] **Step 4: Run final readiness gate**

Run:

```bash
make ready-for-adopters
```

Expected: PASS. This proves lint, isolated migrated tests, and Docker build all pass from the current branch.

- [ ] **Step 5: Check Git state**

Run:

```bash
git status --short
```

Expected: only intended phase 1 files are modified or newly added.

- [ ] **Step 6: Final commit if previous tasks were not committed individually**

If the implementation was done as one batch instead of per-task commits, commit with:

```bash
git add .dockerignore Makefile README.md docs/startup-readiness.md docs/adoption-checklist.md scripts/ci/bootstrap-smoke.sh scripts/seed.sh tests/integration/testhelpers_test.go
git commit -m "fix: make phase 1 readiness gate reliable"
```

## Self-Review

Spec coverage:

- Self-contained readiness gate: Task 1.
- `.env` loading and integration fallback alignment: Task 2.
- Docker build context hygiene: Task 3.
- Seed script correctness: Task 4.
- Documentation honesty: Task 5.
- Final verification: Task 6.

Placeholder scan:

- No unresolved placeholder markers or unspecified edge-case instructions remain.
- Every code-changing step includes exact replacement content.

Type and command consistency:

- `DATABASE_URL` fallback matches `.env.example` and `docker-compose.yml`.
- `ready-for-adopters` relies on `bootstrap-smoke`, and `bootstrap-smoke` now runs race-enabled tests.
- Commands use existing repo targets and file paths.
