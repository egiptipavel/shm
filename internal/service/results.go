package service

import (
	"context"
	"database/sql"
	"errors"
	"shm/internal/config"
	"shm/internal/model"
	"shm/internal/repository"
)

type ResultsService struct {
	results repository.ResultsProvider
	config  config.CommonConfig
}

func NewResultsService(
	results repository.ResultsProvider,
	config config.CommonConfig,
) *ResultsService {
	return &ResultsService{
		results: results,
		config:  config,
	}
}

func (r *ResultsService) AddResult(ctx context.Context, result model.CheckResult) error {
	ctx, cancel := context.WithTimeout(ctx, r.config.DbQueryTimeoutSec)
	defer cancel()

	return r.results.AddResult(ctx, result)
}

func (r *ResultsService) GetNLastResultsForSite(
	ctx context.Context,
	site model.Site,
	number int,
) ([]model.CheckResult, error) {
	ctx, cancel := context.WithTimeout(ctx, r.config.DbQueryTimeoutSec)
	defer cancel()

	return r.results.GetNLastResultsForSite(ctx, site, number)
}

func (r *ResultsService) GetSecondToLastSuccessfulResultForSite(
	ctx context.Context,
	site model.Site,
) (*model.CheckResult, error) {
	ctx, cancel := context.WithTimeout(ctx, r.config.DbQueryTimeoutSec)
	defer cancel()

	successfulResult, err := r.results.GetSecondToLastSuccessfulResultForSite(ctx, site)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &successfulResult, nil
}
