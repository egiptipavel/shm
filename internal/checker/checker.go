package checker

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"shm/internal/broker/rabbitmq"
	"shm/internal/lib/sl"
	"shm/internal/model"
	"shm/internal/repository"
	"sync"
	"syscall"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

type Checker struct {
	broker       *rabbitmq.RabbitMQ
	msgs         <-chan amqp.Delivery
	results      *repository.Results
	sites        *repository.Sites
	intervalMins int
}

func New(db *sql.DB, broker *rabbitmq.RabbitMQ, intervalMins int) (*Checker, error) {
	msgs, err := broker.ConsumeChecks()
	if err != nil {
		return nil, fmt.Errorf("failed to register a consumer for checks: %w", err)
	}
	return &Checker{
		broker:       broker,
		msgs:         msgs,
		results:      repository.NewResultsRepo(db),
		sites:        repository.NewSitesRepo(db),
		intervalMins: intervalMins,
	}, nil
}

func (c *Checker) Start() {
	var wg sync.WaitGroup
	done := make(chan struct{})
	exit := make(chan os.Signal, 1)
	signal.Notify(exit, os.Interrupt, syscall.SIGTERM)

	for range 1000 {
		wg.Add(1)
		go func() {
			defer wg.Done()

			for {
				select {
				case <-done:
					return
				case msg, ok := <-c.msgs:
					if !ok {
						return
					}
					var site model.Site
					err := json.Unmarshal(msg.Body, &site)
					if err != nil {
						slog.Error("failed to parse site", sl.Error(err))
						return
					}
					c.monitorSite(site)
				}
			}
		}()
	}

	s := <-exit
	slog.Info("exit signal was received", slog.String("signal", s.String()))
	close(done)

	wg.Wait()
}

func (c *Checker) monitorSite(site model.Site) {
	result, err := c.checkSite(site)
	if err != nil {
		slog.Error(
			"unsuccessful checking of site",
			slog.String("url", site.Url),
			sl.Error(err),
		)
	} else {
		slog.Info("successful checking of site", sl.CheckResult(result))
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	err = c.results.AddResult(ctx, result)
	cancel()
	if err != nil {
		slog.Error("failed to add check result to storage", sl.Error(err))
	}

	err = c.sendResult(result)
	if err != nil {
		slog.Error("failed to send check result to broker", sl.Error(err))
	}
}

func (c *Checker) checkSite(site model.Site) (result model.CheckResult, err error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
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

func (c *Checker) sendResult(result model.CheckResult) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	body, err := json.Marshal(result)
	if err != nil {
		return fmt.Errorf("failed to marshal result: %w", err)
	}

	return c.broker.PublishToResults(ctx, amqp.Publishing{
		ContentType: "application/json",
		Body:        body,
	})
}
