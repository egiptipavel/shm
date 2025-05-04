-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS check_results (
    site_id INTEGER NOT NULL,
    time TIMESTAMP NOT NULL,
    latency INTEGER,
    code INTEGER
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS check_results;
-- +goose StatementEnd
