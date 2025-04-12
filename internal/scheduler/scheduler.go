package scheduler

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"shm/internal/broker/rabbitmq"
	"shm/internal/lib/logger"
	"shm/internal/model"
	"shm/internal/repository"
	"syscall"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

type Scheduler struct {
	broker       *rabbitmq.RabbitMQ
	sites        *repository.Sites
	intervalMins int
}

func New(db *sql.DB, broker *rabbitmq.RabbitMQ, interval int) *Scheduler {
	return &Scheduler{
		broker:       broker,
		sites:        repository.NewSitesRepo(db),
		intervalMins: interval,
	}
}

func (s *Scheduler) Start() {
	exit := make(chan os.Signal, 1)
	signal.Notify(exit, os.Interrupt, syscall.SIGTERM)

loop:
	for {
		start := time.Now()

		sites, err := s.getSites()
		if err != nil {
			slog.Error("failed to get sites", logger.Error(err))
			break
		}

		for _, site := range sites {
			select {
			case s := <-exit:
				slog.Info("exit signal was received", slog.String("signal", s.String()))
				break loop
			default:
				err = s.sendSite(site)
				if err != nil {
					slog.Error("failed to send site", logger.Error(err))
					break loop
				}
				slog.Info("successfully sending site to queue", logger.Site(site))
			}
		}

		select {
		case <-time.After(time.Duration(s.intervalMins)*time.Minute - time.Since(start)):
		case s := <-exit:
			slog.Info("exit signal was received", slog.String("signal", s.String()))
			break loop
		}
	}
}

func (s *Scheduler) getSites() ([]model.Site, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	return s.sites.GetAllMonitoredSites(ctx)
}

func (s *Scheduler) sendSite(site model.Site) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	body, err := json.Marshal(site)
	if err != nil {
		return fmt.Errorf("failed to marshal site: %w", err)
	}

	return s.broker.PublishToChecks(ctx, amqp.Publishing{
		ContentType: "application/json",
		Body:        body,
	})
}
