package alert

import (
	"context"
	"database/sql"
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
	"shm/internal/repository"
	"syscall"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

type AlertService struct {
	db           *sql.DB
	broker       *rabbitmq.RabbitMQ
	resultsQueue <-chan amqp.Delivery
	resultsRepo  *repository.Results
	config       config.AlertServiceConfig
}

func New(
	db *sql.DB,
	broker *rabbitmq.RabbitMQ,
	config config.AlertServiceConfig,
) (*AlertService, error) {
	resultsQueue, err := broker.ConsumeResults()
	if err != nil {
		return nil, fmt.Errorf("failed to register a consumer for results: %w", err)
	}
	return &AlertService{
		db:           db,
		broker:       broker,
		resultsQueue: resultsQueue,
		resultsRepo:  repository.NewResultsRepo(db),
		config:       config,
	}, nil
}

func (a *AlertService) Start() {
	ctx := context.Background()
	ctx, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := a.routine(ctx); err != nil && !errors.Is(err, context.Canceled) {
		slog.Error("error from alert service", sl.Error(err))
	}
}

func (a *AlertService) routine(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case msg, ok := <-a.resultsQueue:
			if !ok {
				return fmt.Errorf("channel with messages was closed")
			}
			var result model.CheckResult
			if err := json.Unmarshal(msg.Body, &result); err != nil {
				return fmt.Errorf("failed to parse check result: %w", err)
			}
			if err := a.handleResult(ctx, result); err != nil {
				return fmt.Errorf("failed to handle check result: %w", err)
			}
			slog.Info("successful handling of check result", sl.CheckResult(result))
		}
	}
}

func (a *AlertService) handleResult(ctx context.Context, result model.CheckResult) error {
	ctx, cancel := context.WithTimeout(ctx, a.config.DbQueryTimeoutSec)
	lastResults, err := a.resultsRepo.GetLastThreeResultsForSite(ctx, result.Site)
	cancel()
	if err != nil {
		return fmt.Errorf("failed to get last three results for site: %w", err)
	}

	var message string
	switch len(lastResults) {
	case 0:
		return fmt.Errorf("at least one result must exist")
	case 1:
		return nil
	case 2:
		if !lastResults[0].IsSuccessful() && !lastResults[1].IsSuccessful() {
			message = fmt.Sprintf(
				"Bad news. The website %s is temporarily unavailable.",
				result.Site.Url,
			)
		}
	case 3:
		if lastResults[0].IsSuccessful() &&
			!lastResults[1].IsSuccessful() &&
			!lastResults[2].IsSuccessful() {
			ctx, cancel := context.WithTimeout(ctx, a.config.DbQueryTimeoutSec)
			successfulResult, err := a.resultsRepo.GetSecondToLastSuccessfulResultForSite(ctx, result.Site)
			cancel()
			if err != nil && !errors.Is(err, sql.ErrNoRows) {
				return fmt.Errorf("failed to get second to last successful result for site: %w", err)
			}

			if err == nil {
				message = fmt.Sprintf(
					"Good news! The website %s is back up after %d minutes.",
					result.Site.Url,
					int(time.Since(successfulResult.Time).Minutes()),
				)
			} else {
				message = fmt.Sprintf("Good news! The website %s is back up.", result.Site.Url)
			}
		} else if !lastResults[0].IsSuccessful() &&
			!lastResults[1].IsSuccessful() &&
			lastResults[2].IsSuccessful() {
			message = fmt.Sprintf(
				"Bad news. The website %s is temporarily unavailable.",
				result.Site.Url,
			)
		}
	}

	if message != "" {
		notification := model.Notification{
			Url:     result.Site.Url,
			Message: message,
		}
		slog.Info("sending notification", sl.Notification(notification))
		return a.sendNotification(ctx, notification)
	}
	return nil
}

func (a *AlertService) sendNotification(ctx context.Context, notification model.Notification) error {
	body, err := json.Marshal(notification)
	if err != nil {
		return fmt.Errorf("failed to marshal notification: %w", err)
	}

	ctx, cancel := context.WithTimeout(ctx, a.config.BrokerTimeoutSec)
	defer cancel()
	return a.broker.PublishToNotifications(ctx, amqp.Publishing{
		ContentType: "application/json",
		Body:        body,
	})
}
