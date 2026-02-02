-- +goose Up
CREATE TABLE liquidity_pools (
    pool_id TEXT PRIMARY KEY,
    total_shares NUMERIC(20, 7) NOT NULL,
    reserve_a_code TEXT NOT NULL,
    reserve_a_issuer TEXT NOT NULL DEFAULT '',
    reserve_a_amount NUMERIC(20, 7) NOT NULL,
    reserve_b_code TEXT NOT NULL,
    reserve_b_issuer TEXT NOT NULL DEFAULT '',
    reserve_b_amount NUMERIC(20, 7) NOT NULL,
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE account_lp_shares (
    account_id TEXT NOT NULL,
    pool_id TEXT NOT NULL,
    share_balance NUMERIC(20, 7) NOT NULL,
    xlm_value NUMERIC(20, 7),
    PRIMARY KEY (account_id, pool_id)
);

CREATE INDEX idx_lp_shares_account ON account_lp_shares(account_id);

-- +goose Down
DROP TABLE IF EXISTS account_lp_shares;
DROP TABLE IF EXISTS liquidity_pools;
