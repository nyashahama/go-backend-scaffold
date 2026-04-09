-- +goose Up

ALTER TABLE users
    ADD COLUMN token_version INTEGER NOT NULL DEFAULT 0;

UPDATE users
SET email = LOWER(TRIM(email));

CREATE UNIQUE INDEX users_email_lower_unique_idx
    ON users ((LOWER(email)));

-- +goose Down

DROP INDEX IF EXISTS users_email_lower_unique_idx;

ALTER TABLE users
    DROP COLUMN IF EXISTS token_version;
