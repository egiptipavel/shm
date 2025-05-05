package db

import (
	"context"
	"database/sql"
	"shm/internal/repository"
	repo "shm/internal/repository/postgres"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

type Postgres struct {
	db      *sql.DB
	chats   repository.ChatsProvider
	results repository.ResultsProvider
	sites   repository.SitesProvider
}

func NewPostgres(url string) (*Postgres, error) {
	db, err := sql.Open("pgx", url)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err = db.PingContext(ctx); err != nil {
		return nil, err
	}

	return &Postgres{
		db:      db,
		chats:   repo.NewChatsRepo(db),
		results: repo.NewResultsRepo(db),
		sites:   repo.NewSitesRepo(db),
	}, nil
}

func (p *Postgres) DB() *sql.DB {
	return p.db
}

func (p *Postgres) ChatsRepo() repository.ChatsProvider {
	return p.chats
}

func (p *Postgres) ResultsRepo() repository.ResultsProvider {
	return p.results
}

func (p *Postgres) SitesRepo() repository.SitesProvider {
	return p.sites
}

func (p *Postgres) Close() error {
	return p.db.Close()
}
