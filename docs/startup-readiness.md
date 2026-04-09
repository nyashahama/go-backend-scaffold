# Startup Readiness

This scaffold is "ready to hand to a random startup" only when the maintainers can run:

```bash
make ready-for-adopters
```

and the command completes successfully on the current branch.

Use a current local Go toolchain and current project tools before relying on this gate.

## What The Gate Proves

`make ready-for-adopters` is the final local release gate for this repository. It runs:

1. `golangci-lint run ./...`
2. `go test ./... -race`
3. `make bootstrap-smoke`
4. `docker build -t go-backend-scaffold:ready .`

Passing this gate means the current branch has cleared the strongest local checks the scaffold exposes today:

- static analysis passes
- the repository test sweep passes with race detection
- a clean-path bootstrap works without assuming an already prepared local Postgres or Redis setup
- the Docker image still builds

## What The Gate Does Not Prove

This command is intentionally narrow. It does not replace:

- the [adoption checklist](adoption-checklist.md)
- startup-specific production hardening
- environment-specific deployment validation
- ownership changes such as module path, repository naming, registries, and release automation
- legal, privacy, security review, or compliance work

## Maintainer Standard

Use `make ready-for-adopters` before telling adopters that the scaffold is ready for evaluation or handoff.

If the command fails, the scaffold is not ready for that claim yet. Fix the failing check or narrow the claim.
