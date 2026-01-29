-- +goose Up
ALTER TABLE accounts ADD COLUMN council_delegate_to TEXT;
CREATE INDEX idx_accounts_council_delegate ON accounts(council_delegate_to) WHERE council_delegate_to IS NOT NULL;

-- +goose Down
DROP INDEX IF EXISTS idx_accounts_council_delegate;
ALTER TABLE accounts DROP COLUMN IF EXISTS council_delegate_to;
