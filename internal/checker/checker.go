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
	"shm/internal/lib/logger"
	"shm/internal/model"
	"shm/internal/repository"
	"sync"
	"syscall"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

type Checker struct {
	conn         *amqp.Connection
	ch           *amqp.Channel
	msgs         <-chan amqp.Delivery
	results      *repository.Results
	sites        *repository.Sites
	intervalMins int
}

func New(db *sql.DB, intervalMins int) (*Checker, error) {
	conn, err := amqp.Dial("amqp://guest:guest@localhost:5672/")
	if err != nil {
		return nil, fmt.Errorf("failed to connect to RabbitMQ: %w", err)
	}

	ch, err := conn.Channel()
	if err != nil {
		return nil, fmt.Errorf("failed to create channel: %w", err)
	}

	q, err := ch.QueueDeclare(
		"checks", // name
		false,    // durable
		false,    // delete when unused
		false,    // exclusive
		false,    // no-wait
		nil,      // arguments
	)
	if err != nil {
		return nil, fmt.Errorf("failed to declare a queue: %w", err)
	}

	msgs, err := ch.Consume(
		q.Name, // queue
		"",     // consumer
		true,   // auto-ack
		false,  // exclusive
		false,  // no-local
		false,  // no-wait
		nil,    // args
	)
	if err != nil {
		return nil, fmt.Errorf("failed to register a consumer: %w", err)
	}

	return &Checker{
		conn:         conn,
		ch:           ch,
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
						slog.Error("failed to get site", logger.Error(err))
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
			logger.Error(err),
		)
	} else {
		slog.Info("successful checking of site", logger.CheckResult(result))
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	err = c.results.AddResult(ctx, result)
	cancel()
	if err != nil {
		slog.Error("failed to add check result to storage", logger.Error(err))
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

func (c *Checker) Close() {
	c.ch.Close()
	c.conn.Close()
}
