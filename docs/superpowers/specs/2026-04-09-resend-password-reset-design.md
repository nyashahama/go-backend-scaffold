# Resend Password Reset Integration Design

## Context

The scaffold now treats password reset as a production-critical auth capability. Earlier hardening work prevents production startup with a `NoopSender`, but the application still lacks a concrete email provider. The user uses Resend and already maintains environment variables in the following style:

- `RESEND_API_KEY`
- `EMAIL_FROM`
- `EMAIL_FROM_NAME`
- `APP_BASE_URL`

This design adds a Resend-backed implementation for transactional password-reset email without weakening the anti-enumeration properties of the existing auth flow.

## Goals

- Send password-reset emails through Resend using the existing `notification.Sender` abstraction.
- Keep the public `forgot-password` API behavior unchanged: always return success, never reveal account existence.
- Require complete email configuration in production so the service cannot start with a broken reset flow.
- Keep tests deterministic by avoiding live network calls.

## Non-Goals

- Building a generalized multi-provider email framework.
- Adding template storage, localization, or a full notification system.
- Sending any email types other than password reset in this change.

## Recommended Approach

Add a concrete `notification.ResendSender` and keep provider selection fully env-driven. Runtime wiring remains simple:

- If all Resend email variables are configured, instantiate `ResendSender`.
- In non-production environments, allow `NoopSender` as a convenience.
- In production, reject startup unless `RESEND_API_KEY`, `EMAIL_FROM`, `EMAIL_FROM_NAME`, and `APP_BASE_URL` are present and a real sender is active.

This keeps the scaffold easy to understand while making production behavior safe by default.

## Design

### 1. Configuration

Extend `internal/config.Config` with:

- `ResendAPIKey string`
- `EmailFrom string`
- `EmailFromName string`

Load them from:

- `RESEND_API_KEY`
- `EMAIL_FROM`
- `EMAIL_FROM_NAME`

Validation rules:

- Existing global config validation remains responsible for universally required settings such as `DATABASE_URL`, `REDIS_URL`, and `JWT_SECRET`.
- Resend-specific fields are validated at runtime, because they are only required when the service is expected to send email.
- Production startup requires:
  - `APP_BASE_URL`
  - `RESEND_API_KEY`
  - `EMAIL_FROM`
  - `EMAIL_FROM_NAME`
  - a non-`NoopSender`

### 2. Notification Provider

Add a new concrete sender in `internal/notification`, backed by the official Resend Go SDK.

Responsibilities:

- Hold the Resend client and immutable sender metadata.
- Format the outbound `From` header as:
  - `"<EMAIL_FROM_NAME> <EMAIL_FROM>"`
- Implement:
  - `SendPasswordReset(ctx context.Context, to, resetURL string) error`

Email content:

- Subject: `Reset your password`
- HTML body:
  - short explanation
  - reset link
  - note that the link expires

The sender remains narrow on purpose. Template complexity is intentionally deferred.

### 3. Runtime Wiring

`cmd/server/main.go` will:

- Build a `notification.ResendSender` when Resend config is available.
- Fall back to `notification.NoopSender` only in non-production environments.
- Call the existing runtime validation helper so production startup fails fast if email config is incomplete.

This keeps startup behavior explicit and avoids silently shipping a broken password-reset flow.

### 4. Forgot-Password Behavior

The auth handler and service behavior stays externally stable:

- `POST /api/v1/auth/forgot-password` always returns success.
- Unknown email addresses still produce the same success response.
- Valid users still receive a reset email when sending succeeds.

Failure handling:

- If Resend returns an error, the handler still returns success.
- The server logs the failure for operators.
- The response body does not change, preserving anti-enumeration behavior.

### 5. Testing Strategy

Testing stays offline and deterministic.

Add unit tests for:

- config loading of new Resend-related fields
- sender construction and `From` formatting
- runtime validation rules for production email requirements

Add sender tests for:

- missing required config
- correct Resend request payload construction

Do not call Resend in tests.

Existing integration tests for auth flows should continue using test doubles or capture senders. The integration layer verifies auth behavior, not third-party email delivery.

## Error Handling

- Invalid or missing Resend config in production: startup error, process exits.
- Resend send failure during forgot-password: log internally, return public success response.
- Invalid `From` metadata during sender construction: return configuration error before serving traffic.

## Security Notes

- No new enumeration signal is introduced.
- Reset links continue to use the existing hardened token flow.
- The sender does not store or expose provider secrets beyond process memory.
- Production failure is preferred over silent degradation for password reset capability.

## Implementation Outline

1. Add config fields and tests for Resend env vars.
2. Add `notification.ResendSender` plus unit tests.
3. Wire sender selection into `cmd/server/main.go`.
4. Tighten runtime validation to require full Resend config in production.
5. Update `.env.example` to document the new settings.
6. Run full unit and integration verification.

## Open Decisions Resolved

- Provider approach: concrete Resend sender behind the existing interface.
- Production requirements: all three email vars are required:
  - `RESEND_API_KEY`
  - `EMAIL_FROM`
  - `EMAIL_FROM_NAME`
- Public forgot-password behavior: always return success even if sending fails.
