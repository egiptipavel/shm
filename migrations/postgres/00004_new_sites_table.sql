-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS sites (
	id INTEGER PRIMARY KEY GENERATED ALWAYS AS IDENTITY,
	url TEXT UNIQUE NOT NULL
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS sites;
-- +goose StatementEnd
