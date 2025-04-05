package scheduler

import (
	"database/sql"
	"shm/internal/repository"
)

type Scheduler struct {
	sites *repository.Sites
}

func New(db *sql.DB) *Scheduler {
	return &Scheduler{repository.NewSitesRepo(db)}
}

func (s *Scheduler) Start() {
	panic("unimplemented")
}
