package scheduler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
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
)

type Scheduler struct {
	broker *rabbitmq.RabbitMQ
	sites  *service.SitesService
	config config.SchedulerConfig
}

func New(
	broker *rabbitmq.RabbitMQ,
	sites *service.SitesService,
	config config.SchedulerConfig,
) *Scheduler {
	return &Scheduler{
		broker: broker,
		sites:  sites,
		config: config,
	}
}

func (s *Scheduler) Start() {
	ctx := context.Background()
	ctx, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := s.routine(ctx); err != nil && !errors.Is(err, context.Canceled) {
		slog.Error("error from sheduler", sl.Error(err))
	}
}

func (s *Scheduler) routine(ctx context.Context) error {
	t := time.NewTicker(s.config.IntervalMin)
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-t.C:
		}

		sites, err := s.sites.GetAllMonitoredSites(ctx)
		if err != nil {
			return fmt.Errorf("failed to get sites from database: %w", err)
		}

		for _, site := range sites {
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
			}

			if err = s.sendSiteToBroker(ctx, site); err != nil {
				return fmt.Errorf("failed to send site to broker: %w", err)
			}
			slog.Info("successfully sending site to broker", sl.Site(site))
		}
	}
}

func (s *Scheduler) sendSiteToBroker(ctx context.Context, site model.Site) error {
	body, err := json.Marshal(site)
	if err != nil {
		return fmt.Errorf("failed to marshal site: %w", err)
	}

	ctx, cancel := context.WithTimeout(ctx, s.config.BrokerTimeoutSec)
	defer cancel()
	return s.broker.PublishToChecks(ctx, amqp.Publishing{
		ContentType: "application/json",
		Body:        body,
	})
}
