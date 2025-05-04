-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS chat_to_site (
    chat_id INTEGER NOT NULL,
    site_id INTEGER NOT NULL,
    PRIMARY KEY (chat_id, site_id)
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS chat_to_site;
-- +goose StatementEnd
