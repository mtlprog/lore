-- +goose Up
ALTER TABLE accounts ADD COLUMN name TEXT DEFAULT '';

-- +goose Down
ALTER TABLE accounts DROP COLUMN name;
