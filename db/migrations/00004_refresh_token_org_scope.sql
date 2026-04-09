-- +goose Up

ALTER TABLE refresh_tokens
    ADD COLUMN org_id UUID REFERENCES orgs(id) ON DELETE CASCADE;

UPDATE refresh_tokens rt
SET org_id = om.org_id
FROM org_memberships om
WHERE om.user_id = rt.user_id
  AND rt.org_id IS NULL;

ALTER TABLE refresh_tokens
    ALTER COLUMN org_id SET NOT NULL;

CREATE INDEX idx_refresh_tokens_org_id ON refresh_tokens(org_id);

-- +goose Down

DROP INDEX IF EXISTS idx_refresh_tokens_org_id;

ALTER TABLE refresh_tokens
    DROP COLUMN IF EXISTS org_id;
