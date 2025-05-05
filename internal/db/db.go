package db

import (
	"database/sql"
	"shm/internal/repository"
)

type Database interface {
	DB() *sql.DB

	ChatsRepo() repository.ChatsProvider
	ResultsRepo() repository.ResultsProvider
	SitesRepo() repository.SitesProvider

	Close() error
}
