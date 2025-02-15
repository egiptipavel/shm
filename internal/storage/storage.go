package storage

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

const schema = `CREATE TABLE IF NOT EXISTS check_results(
	url VARCHAR(64) NOT NULL,
	time TIMESTAMP NOT NULL,
	latency INTEGER,
	code INTEGER
)`

type ChechResult struct {
	Url     string
	Time    time.Time
	Latency int64
	Code    int
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
	_, err := db.ExecContext(ctx, schema)
	return err
}

func (s *Storage) Add(ctx context.Context, result ChechResult) error {
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

func (s *Storage) GetAll(ctx context.Context) ([]ChechResult, error) {
	rows, err := s.db.QueryContext(ctx, "SELECT * from check_results")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	results := make([]ChechResult, 0)
	for rows.Next() {
		result := ChechResult{}
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

func (s *Storage) Close() {
	s.db.Close()
}
