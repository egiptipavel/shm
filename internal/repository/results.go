package repository

import (
	"context"
	"shm/internal/model"
)

type ResultsProvider interface {
	AddResult(ctx context.Context, result model.CheckResult) error
	GetNLastResultsForSite(ctx context.Context, site model.Site, n int) ([]model.CheckResult, error)
	GetSecondToLastSuccessfulResultForSite(
		ctx context.Context,
		site model.Site,
	) (model.CheckResult, error)
}
