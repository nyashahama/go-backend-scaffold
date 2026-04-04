-- name: CreateUser :one
INSERT INTO users (email, password_hash, full_name)
VALUES ($1, $2, $3)
RETURNING *;

-- name: GetUserByEmail :one
SELECT * FROM users
WHERE email = $1
LIMIT 1;

-- name: GetUserByID :one
SELECT * FROM users
WHERE id = $1
LIMIT 1;

-- name: UpdateUserPassword :exec
UPDATE users
SET password_hash = $2
WHERE id = $1;

-- name: CreateOrg :one
INSERT INTO orgs (name)
VALUES ($1)
RETURNING *;

-- name: GetOrg :one
SELECT * FROM orgs
WHERE id = $1
LIMIT 1;

-- name: CreateOrgMembership :one
INSERT INTO org_memberships (user_id, org_id, role)
VALUES ($1, $2, $3)
RETURNING *;

-- name: GetOrgMembershipByUser :one
SELECT * FROM org_memberships
WHERE user_id = $1 AND org_id = $2
LIMIT 1;

-- name: ListOrgMembershipsByUser :many
SELECT * FROM org_memberships
WHERE user_id = $1
ORDER BY created_at;

-- name: CreateRefreshToken :one
INSERT INTO refresh_tokens (token, user_id, expires_at)
VALUES ($1, $2, $3)
RETURNING *;

-- name: GetRefreshToken :one
SELECT * FROM refresh_tokens
WHERE token = $1
  AND revoked = FALSE
  AND expires_at > NOW()
LIMIT 1;

-- name: RevokeRefreshToken :exec
UPDATE refresh_tokens
SET revoked = TRUE
WHERE token = $1;

-- name: RevokeAllUserRefreshTokens :exec
UPDATE refresh_tokens
SET revoked = TRUE
WHERE user_id = $1;
