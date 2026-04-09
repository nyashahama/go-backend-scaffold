# Auth And Runtime Hardening Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Eliminate the auth correctness bugs, harden request/runtime middleware, and repair the basic developer bootstrap path for this scaffold.

**Architecture:** Keep the current project shape, but tighten correctness at the service and middleware boundaries. The main changes are atomic token consumption in auth flows, stricter request parsing, safer HTTP writer wrapping, and clearer bootstrap/test assets.

**Tech Stack:** Go, chi, pgx/sqlc, Redis, Prometheus, slog, Docker

---

### Task 1: Make Auth Token Consumption Atomic

**Files:**
- Modify: `db/queries/auth.sql`
- Modify: `db/gen/auth.sql.go`
- Modify: `internal/auth/service.go`
- Test: `internal/auth/service_test.go`

- [ ] Add failing tests for refresh-token reuse races and password-reset single-use behavior.
- [ ] Regenerate or patch the sqlc query surface to support atomic refresh-token consumption.
- [ ] Implement transactional refresh rotation using a consume-and-reissue flow.
- [ ] Implement atomic password-reset token consumption before password mutation.
- [ ] Run focused auth tests, then full unit tests.

### Task 2: Fix Auth Error Semantics And Input Strictness

**Files:**
- Modify: `internal/auth/service.go`
- Modify: `internal/auth/handler.go`
- Test: `internal/auth/service_test.go`
- Test: `internal/auth/handler_test.go`

- [ ] Add failing tests proving DB failures are not returned as auth failures and malformed JSON is rejected strictly.
- [ ] Return `ErrInvalidCredentials` and `ErrInvalidToken` only for real user-auth cases.
- [ ] Normalize inbound email values consistently on register/login/forgot-password.
- [ ] Tighten JSON decoding to reject unknown fields and trailing data.
- [ ] Run focused auth tests, then full unit tests.

### Task 3: Harden Middleware Behavior

**Files:**
- Modify: `internal/middleware/logging.go`
- Modify: `internal/middleware/metrics.go`
- Modify: `internal/middleware/ratelimit.go`
- Modify: `internal/server/router.go`
- Test: `internal/middleware/logging_test.go`
- Test: `internal/middleware/ratelimit_test.go`
- Test: `internal/server/router_test.go`

- [ ] Add failing tests for preserved optional writer interfaces, rate-limit key behavior, and unthrottled health/metrics endpoints.
- [ ] Replace the response-writer wrapper with one that preserves supported optional interfaces.
- [ ] Make rate limiting proxy-aware and Redis-atomic, using request context rather than background context.
- [ ] Exempt health/readiness/metrics endpoints from normal API rate limiting.
- [ ] Run focused middleware/server tests, then full unit tests.

### Task 4: Repair Bootstrap And Documentation Gaps

**Files:**
- Add: `.env.example`
- Modify: `README.md`
- Test: manual verification through documented commands

- [ ] Add a working `.env.example` aligned with `docker-compose.yml`.
- [ ] Update README bootstrap steps to reflect the actual repo contents and required environment.
- [ ] Verify the documented quickstart is internally consistent.
