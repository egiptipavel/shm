package checker

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"shm/internal/broker"
	"shm/internal/config"
	"shm/internal/lib/sl"
	"shm/internal/model"
	"shm/internal/service"
	"syscall"
	"time"

	"golang.org/x/sync/errgroup"
)

type Checker struct {
	broker         broker.MessageBroker
	resultsService *service.ResultsService
	sitesService   *service.SitesService
	config         config.CheckerConfig
}

func New(
	broker broker.MessageBroker,
	resultsService *service.ResultsService,
	sitesService *service.SitesService,
	config config.CheckerConfig,
) *Checker {
	return &Checker{
		broker:         broker,
		resultsService: resultsService,
		sitesService:   sitesService,
		config:         config,
	}
}

func (c *Checker) Start() {
	ctx := context.Background()
	ctx, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer stop()

	sitesQueue, err := c.broker.ConsumeSites(ctx)
	if err != nil {
		slog.Error("failed to register a consumer for sites", sl.Error(err))
		return
	}

	g, ctx := errgroup.WithContext(ctx)

	for range c.config.Workers {
		g.Go(func() error {
			return c.workerRoutine(ctx, sitesQueue)
		})
	}

	if err := g.Wait(); err != nil && !errors.Is(err, context.Canceled) {
		slog.Error("error from worker", sl.Error(err))
	}
}

func (c *Checker) workerRoutine(ctx context.Context, sitesQueue <-chan model.Site) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case site, ok := <-sitesQueue:
			if !ok {
				return fmt.Errorf("queue with sites was closed")
			}
			if err := c.monitorSite(ctx, site); err != nil {
				return fmt.Errorf("failed to monitor site: %w", err)
			}
		}
	}
}

func (c *Checker) monitorSite(ctx context.Context, site model.Site) error {
	result, err := c.checkSite(ctx, site)
	if err != nil {
		slog.Error("unsuccessful checking of site", slog.String("url", site.Url), sl.Error(err))
	} else {
		slog.Info("successful checking of site", sl.CheckResult(result))
	}

	if err = c.resultsService.AddResult(ctx, result); err != nil {
		return fmt.Errorf("failed to send check result to database: %w", err)
	}

	if err = c.broker.PublishResult(ctx, result); err != nil {
		return fmt.Errorf("failed to send check result to broker: %w", err)
	}

	return nil
}

func (c *Checker) checkSite(
	ctx context.Context,
	site model.Site,
) (result model.CheckResult, err error) {
	ctx, cancel := context.WithTimeout(ctx, c.config.SiteResponseTimeoutSec)
	defer cancel()

	start := time.Now()
	req, _ := http.NewRequestWithContext(ctx, "GET", site.Url, nil)
	resp, err := http.DefaultClient.Do(req)
	latency := time.Since(start).Milliseconds()
	if err != nil {
		return model.CheckResult{
			Site:    site,
			Time:    start,
			Latency: sql.NullInt64{},
			Code:    sql.NullInt64{},
		}, err
	}
	defer resp.Body.Close()

	return model.CheckResult{
		Site: site,
		Time: start,
		Latency: sql.NullInt64{
			Int64: latency,
			Valid: true,
		},
		Code: sql.NullInt64{
			Int64: int64(resp.StatusCode),
			Valid: true,
		},
	}, nil
}
