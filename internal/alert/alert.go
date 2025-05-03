package alert

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

type AlertService struct {
	broker         *rabbitmq.RabbitMQ
	resultsQueue   <-chan amqp.Delivery
	resultsService *service.ResultsService
	config         config.AlertServiceConfig
}

func New(
	broker *rabbitmq.RabbitMQ,
	results *service.ResultsService,
	config config.AlertServiceConfig,
) (*AlertService, error) {
	if config.NumberOrFailedChecks < 1 {
		return nil, fmt.Errorf("number of failed checks must be at least 1")
	}
	resultsQueue, err := broker.ConsumeResults()
	if err != nil {
		return nil, fmt.Errorf("failed to register a consumer for results: %w", err)
	}
	return &AlertService{
		broker:         broker,
		resultsQueue:   resultsQueue,
		resultsService: results,
		config:         config,
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
			if err := a.sendNotificationIfNeeded(ctx, result.Site); err != nil {
				return fmt.Errorf("failed to handle check result: %w", err)
			}
			slog.Info("successful handling of check result", sl.CheckResult(result))
		}
	}
}

func (a *AlertService) sendNotificationIfNeeded(ctx context.Context, site model.Site) error {
	lastResults, err := a.resultsService.GetNLastResultsForSite(
		ctx, site, a.config.NumberOrFailedChecks+1,
	)
	if err != nil {
		return fmt.Errorf("failed to get last results for site: %w", err)
	}

	if len(lastResults) < a.config.NumberOrFailedChecks {
		slog.Info(
			"number of last results is not enough",
			sl.Site(site),
			slog.Int("last_results", len(lastResults)),
		)
		return nil
	}

	var message string
	if len(lastResults) == a.config.NumberOrFailedChecks && a.allChecksFailed(lastResults) {
		slog.Info("all checks failed", sl.Site(site))
		message = fmt.Sprintf(
			"Bad news. The website %s is temporarily unavailable.",
			site.Url,
		)
	}

	if len(lastResults) == a.config.NumberOrFailedChecks+1 {
		if lastResults[0].IsSuccessful() && a.allChecksFailed(lastResults[1:]) {
			slog.Info("website is back up", sl.Site(site))

			successfulResult, err := a.resultsService.GetSecondToLastSuccessfulResultForSite(ctx, site)
			if err != nil {
				return fmt.Errorf("failed to get second to last successful result for site: %w", err)
			}

			if successfulResult != nil {
				slog.Info("second to last successful result was found", sl.Site(site))
				message = fmt.Sprintf(
					"Good news! The website %s is back up after %d minutes.",
					site.Url,
					int(time.Since(successfulResult.Time).Minutes()),
				)
			} else {
				slog.Info("second to last successful result was not found", sl.Site(site))
				message = fmt.Sprintf("Good news! The website %s is back up.", site.Url)
			}
		} else if a.allChecksFailed(lastResults[:a.config.NumberOrFailedChecks]) &&
			lastResults[len(lastResults)-1].IsSuccessful() {
			message = fmt.Sprintf(
				"Bad news. The website %s is temporarily unavailable.",
				site.Url,
			)
		}

	}

	if message != "" {
		notification := model.Notification{
			Url:     site.Url,
			Message: message,
		}
		slog.Info("sending notification", sl.Notification(notification))
		return a.sendNotification(ctx, notification)
	}
	return nil
}

func (a *AlertService) allChecksFailed(results []model.CheckResult) bool {
	for _, res := range results {
		if res.IsSuccessful() {
			return false
		}
	}
	return true
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
