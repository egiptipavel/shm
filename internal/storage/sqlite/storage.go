package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/mattn/go-sqlite3"
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
	is_subscribed BOOLEAN CHECK (is_subscribed IN (0, 1)) 
)`

const chatToSiteScheme = `
CREATE TABLE IF NOT EXISTS chat_to_site(
	chat_id INTEGER NOT NULL,
	site_id INTEGER NOT NULL,
	PRIMARY KEY(chat_id, site_id)
)`

const sitesScheme = `
CREATE TABLE IF NOT EXISTS sites(
	id INTEGER PRIMARY KEY,
	url TEXT UNIQUE NOT NULL
)`

func New(dataSourceName string) (*sql.DB, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	db, err := connectToDB(ctx, dataSourceName)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %s", err)
	}

	err = initDB(ctx, db)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize database: %s", err)
	}

	return db, nil
}

func connectToDB(ctx context.Context, dataSourceName string) (*sql.DB, error) {
	db, err := sql.Open("sqlite3", dataSourceName)
	if err != nil {
		return nil, err
	}

	if err := db.PingContext(ctx); err != nil {
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
