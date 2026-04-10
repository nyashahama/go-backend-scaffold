# Adoption Checklist

Use this scaffold as a starting point, not a production-ready drop-in. Complete the checklist below before the first production deploy.

## 1. Repository Initialization

- Run `bash scripts/init-template.sh <your-module-path>` once in a fresh, clean clone before making project-specific edits.
- Confirm `go.mod` and internal imports now point at your module path instead of the scaffold's original module path.
- Rename the repository, binary/image tags, and any remaining project-facing labels that should match your company or product.
- Update release automation, container registry ownership, and image names so they publish to your organization rather than the scaffold author's defaults.

## 2. Secrets And Environment

- Replace every example or placeholder secret before shared environments or production use.
- Set a strong `JWT_SECRET` and store it in your secret manager rather than committing it.
- Review `.env.example` and remove variables your startup will not support, or document the ones you add.
- If you deploy behind a reverse proxy or ingress, set `TRUST_PROXY_HEADERS=true` only after you restrict `TRUSTED_PROXY_CIDRS` to the proxy networks you actually trust.

## 3. Data Stores And Migrations

- Provision Postgres and Redis for each environment you operate.
- Run migrations against a fresh database and verify rollback expectations before production deploys.
- Define backup, restore, and retention procedures for Postgres before storing customer data.

## 4. Auth And External Services

- Decide how accounts are created and whether self-serve registration should stay enabled.
- Configure a real email sender for password reset flows, or explicitly disable those endpoints until it exists.
- Review token expiry, CORS, and `APP_BASE_URL` settings so they match your deployed clients.
- Review the default auth rate limits and tighten or replace them if your product has higher-risk login or reset flows.

## 5. Delivery And Operations

- Run `make test-ci` and `make bootstrap-smoke` on your branch before treating the template as adopted.
- Run `make security-check` and review any findings before production release.
- Add CI/CD for your own repository, container registry, and deployment target.
- Verify health checks, logs, metrics, and alerting are wired into your runtime environment.

## 6. Product-Specific Hardening

- Remove scaffold code, routes, and dependencies your product will not use.
- Add authorization rules, rate limits, and audit expectations that match your domain.
- Review legal, privacy, and compliance requirements separately; this scaffold does not satisfy them for you.
