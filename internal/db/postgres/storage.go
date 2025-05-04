package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"shm/internal/config"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

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

	return db, nil
}
