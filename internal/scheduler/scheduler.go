package scheduler

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"shm/internal/broker"
	"shm/internal/config"
	"shm/internal/lib/sl"
	"shm/internal/service"
	"syscall"
	"time"
)

type Scheduler struct {
	broker broker.MessageBroker
	sites  *service.SitesService
	config config.SchedulerConfig
}

func New(
	broker broker.MessageBroker,
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

			if err = s.broker.PublishSite(ctx, site); err != nil {
				return fmt.Errorf("failed to send site to broker: %w", err)
			}
			slog.Info("successfully sending site to broker", sl.Site(site))
		}
	}
}
