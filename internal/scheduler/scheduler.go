package scheduler

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"shm/internal/lib/logger"
	"shm/internal/model"
	"shm/internal/repository"
	"syscall"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

type Scheduler struct {
	conn         *amqp.Connection
	ch           *amqp.Channel
	q            amqp.Queue
	sites        *repository.Sites
	intervalMins int
}

func New(db *sql.DB, interval int) (*Scheduler, error) {
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

	return &Scheduler{
		conn:         conn,
		ch:           ch,
		q:            q,
		sites:        repository.NewSitesRepo(db),
		intervalMins: interval,
	}, nil
}

func (s *Scheduler) Start() {
	exit := make(chan os.Signal, 1)
	signal.Notify(exit, os.Interrupt, syscall.SIGTERM)

loop2:
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
				break loop2
			default:
				err = s.sendSite(site)
				if err != nil {
					slog.Error("failed to send site", logger.Error(err))
					break loop2
				}
				slog.Info("successfully sending site to queue", logger.Site(site))
			}
		}

		select {
		case <-time.After(time.Duration(s.intervalMins)*time.Minute - time.Since(start)):
		case s := <-exit:
			slog.Info("exit signal was received", slog.String("signal", s.String()))
			break loop2
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

	return s.ch.PublishWithContext(ctx, "", s.q.Name, false, false, amqp.Publishing{
		ContentType: "application/json",
		Body:        body,
	})
}

func (s *Scheduler) Close() {
	s.ch.Close()
	s.conn.Close()
}
