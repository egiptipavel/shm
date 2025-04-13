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
	"shm/internal/lib/logger"
	"shm/internal/model"
	"shm/internal/repository"
	"syscall"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

type AlertService struct {
	db      *sql.DB
	broker  *rabbitmq.RabbitMQ
	results *repository.Results
}

func New(db *sql.DB, broker *rabbitmq.RabbitMQ) *AlertService {
	return &AlertService{
		db:      db,
		broker:  broker,
		results: repository.NewResultsRepo(db),
	}
}

func (a *AlertService) Start() error {
	results, err := a.broker.ConsumeResults()
	if err != nil {
		return fmt.Errorf("failed to register a consumer for results: %w", err)
	}

	exit := make(chan os.Signal, 1)
	signal.Notify(exit, os.Interrupt, syscall.SIGTERM)

	a.handleResults(results, exit)

	return nil
}

func (a *AlertService) handleResults(results <-chan amqp.Delivery, exit <-chan os.Signal) {
	for {
		select {
		case s := <-exit:
			slog.Info("exit signal was received", slog.String("signal", s.String()))
			return
		case msg, ok := <-results:
			if !ok {
				return
			}
			var result model.CheckResult
			if err := json.Unmarshal(msg.Body, &result); err != nil {
				slog.Error("failed to parse check result", logger.Error(err))
				return
			}
			if err := a.handleResult(result); err != nil {
				slog.Error("failed to handle result", logger.CheckResult(result), logger.Error(err))
				return
			}
		}
	}
}

func (a *AlertService) handleResult(result model.CheckResult) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	lastResults, err := a.results.GetLastThreeResultsForSite(ctx, result.Site)
	cancel()
	if err != nil {
		return fmt.Errorf("failed to get last two results for site: %w", err)
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
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			successfulResult, err := a.results.GetSecondToLastSuccessfulResultForSite(ctx, result.Site)
			cancel()
			if err != nil && !errors.Is(err, sql.ErrNoRows) {
				return fmt.Errorf("failed to get last successful result for site: %w", err)
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
		return a.sendNotification(model.Notification{
			Url:     result.Site.Url,
			Message: message,
		})
	}
	return nil
}

func (a *AlertService) sendNotification(notif model.Notification) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	body, err := json.Marshal(notif)
	if err != nil {
		return fmt.Errorf("failed to marshal notification: %w", err)
	}

	return a.broker.PublishToNotifications(ctx, amqp.Publishing{
		ContentType: "application/json",
		Body:        body,
	})
}
