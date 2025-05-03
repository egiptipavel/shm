package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"shm/internal/config"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

const checkResultsScheme = `
CREATE TABLE IF NOT EXISTS check_results(
	site_id INTEGER NOT NULL,
	time TIMESTAMP NOT NULL,
	latency INTEGER,
	code INTEGER
)`

const chatsScheme = `
CREATE TABLE IF NOT EXISTS chats(
	id INTEGER PRIMARY KEY,
	is_subscribed BOOLEAN
)`

const chatToSiteScheme = `
CREATE TABLE IF NOT EXISTS chat_to_site(
	chat_id INTEGER NOT NULL,
	site_id INTEGER NOT NULL,
	PRIMARY KEY(chat_id, site_id)
)`

const sitesScheme = `
CREATE TABLE IF NOT EXISTS sites(
	id INTEGER PRIMARY KEY GENERATED ALWAYS AS IDENTITY,
	url TEXT UNIQUE NOT NULL
)`

func New(config config.PostgreSQLConfig) (*sql.DB, error) {
	url := fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s",
		config.User, config.Pass, config.Host, config.Port, config.Db,
	)
	db, err := sql.Open("pgx", url)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err = db.PingContext(ctx); err != nil {
		return nil, err
	}

	if err = initDB(ctx, db); err != nil {
		return nil, err
	}

	return db, nil
}

func initDB(ctx context.Context, db *sql.DB) error {
	if _, err := db.ExecContext(ctx, checkResultsScheme); err != nil {
		return err
	}

	if _, err := db.ExecContext(ctx, chatsScheme); err != nil {
		return err
	}

	if _, err := db.ExecContext(ctx, chatToSiteScheme); err != nil {
		return err
	}

	if _, err := db.ExecContext(ctx, sitesScheme); err != nil {
		return err
	}

	return nil
}
