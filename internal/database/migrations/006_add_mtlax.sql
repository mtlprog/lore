-- +goose Up
ALTER TABLE accounts ADD COLUMN mtlax_balance NUMERIC(20, 7) DEFAULT NULL;
CREATE INDEX idx_accounts_mtlax ON accounts(mtlax_balance) WHERE mtlax_balance IS NOT NULL;

-- +goose Down
DROP INDEX IF EXISTS idx_accounts_mtlax;
ALTER TABLE accounts DROP COLUMN IF EXISTS mtlax_balance;
