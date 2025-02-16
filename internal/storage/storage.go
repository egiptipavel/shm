package storage

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

const resultsSchema = `CREATE TABLE IF NOT EXISTS check_results(
	url VARCHAR(64) NOT NULL,
	time TIMESTAMP NOT NULL,
	latency INTEGER,
	code INTEGER
)`

const subscribersShema = `CREATE TABLE IF NOT EXISTS subscribers(chat_id INTEGER PRIMARY KEY)`

type CheckResult struct {
	Url     string
	Time    time.Time
	Latency int64
	Code    int
}

type Subscriber struct {
	ChatID int64
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
	_, err := db.ExecContext(ctx, resultsSchema)
	if err != nil {
		return err
	}
	_, err = db.ExecContext(ctx, subscribersShema)
	return err
}

func (s *Storage) AddResult(ctx context.Context, result CheckResult) error {
	resultLatency := &result.Latency
	if *resultLatency == 0 {
		resultLatency = nil
	}
	resultCode := &result.Code
	if *resultCode == 0 {
		resultCode = nil
	}
	_, err := s.db.ExecContext(
		ctx,
		"INSERT INTO check_results (url, time, latency, code) VALUES (?, ?, ?, ?)",
		result.Url, result.Time, resultLatency, resultCode,
	)
	return err
}

func (s *Storage) GetAllResults(ctx context.Context) ([]CheckResult, error) {
	rows, err := s.db.QueryContext(ctx, "SELECT * from check_results")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	results := make([]CheckResult, 0)
	for rows.Next() {
		result := CheckResult{}
		err = rows.Scan(&result.Url, &result.Time, &result.Latency, &result.Code)
		if err != nil {
			return nil, err
		}
		results = append(results, result)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return results, nil
}

func (s *Storage) AddSubscriber(ctx context.Context, subscriber Subscriber) error {
	_, err := s.db.ExecContext(
		ctx,
		"INSERT INTO subscribers (chat_id) VALUES (?)",
		subscriber.ChatID,
	)
	return err
}

func (s *Storage) DeleteSubscriber(ctx context.Context, chatId int64) error {
	_, err := s.db.ExecContext(
		ctx,
		"DELETE FROM subscribers WHERE chat_id = ?",
		chatId,
	)
	return err
}

func (s *Storage) GetAllSubscribers(ctx context.Context) ([]Subscriber, error) {
	rows, err := s.db.QueryContext(ctx, "SELECT * from subscribers")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	subscribers := make([]Subscriber, 0)
	for rows.Next() {
		subscriber := Subscriber{}
		err = rows.Scan(&subscriber.ChatID)
		if err != nil {
			return nil, err
		}
		subscribers = append(subscribers, subscriber)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return subscribers, nil
}

func (s *Storage) Close() {
	s.db.Close()
}
