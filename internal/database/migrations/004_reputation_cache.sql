-- +goose Up

-- Reputation scores table (pre-calculated during sync)
CREATE TABLE reputation_scores (
    account_id TEXT PRIMARY KEY,
    weighted_score NUMERIC(10, 4) NOT NULL DEFAULT 0,
    base_score NUMERIC(10, 4) NOT NULL DEFAULT 0,      -- Simple weighted avg (A=4, B=3, C=2, D=1)
    rating_count_a INT NOT NULL DEFAULT 0,
    rating_count_b INT NOT NULL DEFAULT 0,
    rating_count_c INT NOT NULL DEFAULT 0,
    rating_count_d INT NOT NULL DEFAULT 0,
    total_ratings INT NOT NULL DEFAULT 0,
    total_weight NUMERIC(10, 4) NOT NULL DEFAULT 0,    -- Sum of rater weights
    calculated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT valid_weighted_score CHECK (weighted_score >= 0 AND weighted_score <= 4),
    CONSTRAINT valid_base_score CHECK (base_score >= 0 AND base_score <= 4),
    FOREIGN KEY (account_id) REFERENCES accounts(account_id) ON DELETE CASCADE
);

-- Index for sorting by reputation score
CREATE INDEX idx_reputation_scores_weighted ON reputation_scores(weighted_score DESC)
    WHERE total_ratings >= 3;

-- Index for finding accounts with reputation
CREATE INDEX idx_reputation_scores_total ON reputation_scores(total_ratings DESC);

-- +goose Down

DROP TABLE IF EXISTS reputation_scores;
