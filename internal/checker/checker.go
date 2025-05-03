package checker

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"shm/internal/broker/rabbitmq"
	"shm/internal/config"
	"shm/internal/lib/sl"
	"shm/internal/model"
	"shm/internal/service"
	"syscall"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	"golang.org/x/sync/errgroup"
)

type Checker struct {
	broker  *rabbitmq.RabbitMQ
	msgs    <-chan amqp.Delivery
	results *service.ResultsService
	sites   *service.SitesService
	config  config.CheckerConfig
}

func New(
	broker *rabbitmq.RabbitMQ,
	results *service.ResultsService,
	sites *service.SitesService,
	config config.CheckerConfig,
) (*Checker, error) {
	msgs, err := broker.ConsumeChecks()
	if err != nil {
		return nil, fmt.Errorf("failed to register a consumer for checks: %w", err)
	}
	return &Checker{
		broker:  broker,
		msgs:    msgs,
		results: results,
		sites:   sites,
		config:  config,
	}, nil
}

func (c *Checker) Start() {
	ctx := context.Background()
	ctx, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer stop()
	g, ctx := errgroup.WithContext(ctx)

	for range c.config.Workers {
		g.Go(func() error {
			return c.workerRoutine(ctx)
		})
	}

	if err := g.Wait(); err != nil && !errors.Is(err, context.Canceled) {
		slog.Error("error from worker", sl.Error(err))
	}
}

func (c *Checker) workerRoutine(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case msg, ok := <-c.msgs:
			if !ok {
				return fmt.Errorf("channel with messages was closed")
			}
			var site model.Site
			if err := json.Unmarshal(msg.Body, &site); err != nil {
				return fmt.Errorf("failed to parse site: %w", err)
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

	if err = c.results.AddResult(ctx, result); err != nil {
		return fmt.Errorf("failed to send check result to database: %w", err)
	}

	if err = c.sendResultToBroker(ctx, result); err != nil {
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

func (c *Checker) sendResultToBroker(ctx context.Context, result model.CheckResult) error {
	body, err := json.Marshal(result)
	if err != nil {
		return fmt.Errorf("failed to encode check result: %w", err)
	}

	ctx, cancel := context.WithTimeout(ctx, c.config.BrokerTimeoutSec)
	defer cancel()
	return c.broker.PublishToResults(ctx, amqp.Publishing{
		ContentType: "application/json",
		Body:        body,
	})
}
