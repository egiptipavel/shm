package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

const checkResultsScheme = `CREATE TABLE IF NOT EXISTS check_results(
	site_id INTEGER NOT NULL,
	time TIMESTAMP NOT NULL,
	latency INTEGER,
	code INTEGER
)`

const chatsScheme = `CREATE TABLE IF NOT EXISTS chats(
	id INTEGER PRIMARY KEY,
	is_subscribed BOOLEAN CHECK (is_subscribed IN (0, 1)) 
)`

const chatToSiteScheme = `CREATE TABLE IF NOT EXISTS chat_to_site(
	chat_id INTEGER NOT NULL,
	site_id INTEGER NOT NULL,
	PRIMARY KEY(chat_id, site_id)
)`

const sitesScheme = `CREATE TABLE IF NOT EXISTS sites(
	id INTEGER PRIMARY KEY,
	url TEXT UNIQUE NOT NULL
)`

type CheckResult struct {
	Site    Site
	Time    time.Time
	Latency sql.NullInt64
	Code    sql.NullInt64
}

func (c *CheckResult) IsSuccessful() bool {
	return c.Code.Valid && c.Code.Int64 == 200
}

type Chat struct {
	Id           int64
	IsSubscribed bool
}

type ChatToSite struct {
	ChatID int64
	SiteID int64
}

type Site struct {
	Id  int64
	Url string
}

type Storage struct {
	db *sql.DB
}

func NewStorage(dataSourceName string) (*Storage, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	db, err := connectDB(ctx, dataSourceName)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %s", err)
	}

	err = initDB(ctx, db)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize database: %s", err)
	}

	return &Storage{db}, nil
}

func connectDB(ctx context.Context, dataSourceName string) (*sql.DB, error) {
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

func (s *Storage) AddResult(ctx context.Context, result CheckResult) error {
	_, err := s.db.ExecContext(
		ctx,
		"INSERT INTO check_results (site_id, time, latency, code) VALUES (?, ?, ?, ?)",
		result.Site.Id, result.Time, result.Latency, result.Code,
	)

	return err
}

func (s *Storage) GetLastTwoResultsForSite(ctx context.Context, site Site) ([]CheckResult, error) {
	rows, err := s.db.QueryContext(
		ctx,
		`SELECT s.id, s.url, c.time, c.latency, c.code
		FROM check_results AS c
		JOIN sites AS s
		ON c.site_id = s.id
		WHERE s.id = ?
		ORDER BY c.time DESC
		LIMIT 2`,
		site.Id,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []CheckResult
	for rows.Next() {
		var result CheckResult

		err = rows.Scan(
			&result.Site.Id,
			&result.Site.Url,
			&result.Time,
			&result.Latency,
			&result.Code,
		)
		if err != nil {
			return nil, err
		}

		results = append(results, result)
	}

	return results, nil
}

func (s *Storage) GetLastSuccessfulResultForSite(ctx context.Context, site Site) (CheckResult, error) {
	var result CheckResult
	err := s.db.QueryRowContext(ctx,
		`SELECT s.id, s.url, c.time, c.latency, c.code
		FROM check_results as c
		JOIN sites as s
		ON c.site_id = s.id
		WHERE s.id = ? AND c.code = 200
		ORDER BY c.time DESC
		LIMIT 1`,
		site.Id,
	).Scan(
		&result.Site.Id,
		&result.Site.Url,
		&result.Time,
		&result.Latency,
		&result.Code,
	)
	return result, err
}

func (s *Storage) AddChat(ctx context.Context, chat Chat) error {
	_, err := s.db.ExecContext(
		ctx,
		`INSERT INTO chats (id, is_subscribed) 
		VALUES (?, TRUE) 
		ON CONFLICT (id) DO UPDATE SET is_subscribed = TRUE`,
		chat.Id,
	)
	return err
}

func (s *Storage) UpdateChat(ctx context.Context, chatId int64, isSub bool) error {
	_, err := s.db.ExecContext(
		ctx,
		"UPDATE chats SET is_subscribed = ? WHERE id = ?",
		isSub, chatId,
	)
	return err
}

func (s *Storage) GetAllSubscribedOnSiteChats(ctx context.Context, url string) ([]Chat, error) {
	rows, err := s.db.QueryContext(
		ctx,
		`SELECT c.id
		FROM chats as c
		JOIN chat_to_site as cs
		ON c.id = cs.chat_id
		JOIN sites as s
		ON cs.site_id = s.id
		WHERE c.is_subscribed = TRUE AND s.url = ?`,
		url,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var chats []Chat
	for rows.Next() {
		var chat Chat

		err = rows.Scan(&chat.Id)
		if err != nil {
			return nil, err
		}

		chats = append(chats, chat)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return chats, nil
}

func (s *Storage) AddSite(ctx context.Context, chatId int64, url string) error {
	_, err := s.db.ExecContext(
		ctx,
		"INSERT INTO sites (url) VALUES (?) ON CONFLICT DO NOTHING",
		url,
	)
	if err != nil {
		return err
	}

	var siteId int64
	err = s.db.QueryRowContext(ctx, "SELECT id FROM sites WHERE url = ?", url).Scan(&siteId)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil
		}
		return err
	}

	_, err = s.db.ExecContext(
		ctx,
		"INSERT INTO chat_to_site (chat_id, site_id) VALUES (?, ?) ON CONFLICT DO NOTHING",
		chatId, siteId,
	)
	return err
}

func (s *Storage) DeleteSite(ctx context.Context, chatId int64, url string) error {
	var siteId int64
	err := s.db.QueryRowContext(ctx, "SELECT id FROM sites WHERE url = ?", url).Scan(&siteId)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil
		}
		return err
	}

	_, err = s.db.ExecContext(
		ctx,
		"DELETE FROM chat_to_site WHERE chat_id = ? AND site_id = ?",
		chatId, siteId,
	)
	return err
}

func (s *Storage) GetAllMonitoredSites(ctx context.Context) ([]Site, error) {
	rows, err := s.db.QueryContext(
		ctx,
		"SELECT DISTINCT s.id, s.url FROM sites AS s JOIN chat_to_site AS c ON s.id = c.site_id",
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sites []Site
	for rows.Next() {
		var site Site

		err = rows.Scan(&site.Id, &site.Url)
		if err != nil {
			return nil, err
		}

		sites = append(sites, site)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return sites, nil
}

func (s *Storage) GetAllSitesByChatId(ctx context.Context, chatId int64) ([]Site, error) {
	rows, err := s.db.QueryContext(
		ctx,
		`SELECT s.id, s.url 
		FROM chat_to_site as c
		JOIN sites as s
		ON c.site_id = s.id
		WHERE c.chat_id = ?`,
		chatId,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sites []Site
	for rows.Next() {
		var site Site

		err = rows.Scan(&site.Id, &site.Url)
		if err != nil {
			return nil, err
		}

		sites = append(sites, site)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return sites, nil
}

func (s *Storage) Close() {
	s.db.Close()
}
