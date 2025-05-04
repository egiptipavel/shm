-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS chats (id INTEGER PRIMARY KEY, is_subscribed BOOLEAN);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS chats;
-- +goose StatementEnd
