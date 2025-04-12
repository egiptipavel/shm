package telegram

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
	urlpkg "shm/internal/lib/url"
	"shm/internal/model"
	"shm/internal/repository"
	"strings"
	"sync"
	"syscall"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"

	"gopkg.in/telebot.v4"
)

type TGBot struct {
	bot     *telebot.Bot
	broker  *rabbitmq.RabbitMQ
	chats   *repository.Chats
	results *repository.Results
	sites   *repository.Sites
}

func New(token string, db *sql.DB, broker *rabbitmq.RabbitMQ) (*TGBot, error) {
	bot, err := telebot.NewBot(telebot.Settings{
		Token:  token,
		Poller: &telebot.LongPoller{Timeout: 10 * time.Second},
	})
	if err != nil {
		return nil, err
	}

	t := &TGBot{
		bot:     bot,
		broker:  broker,
		chats:   repository.NewChatsRepo(db),
		results: repository.NewResultsRepo(db),
		sites:   repository.NewSitesRepo(db),
	}

	bot.Handle("/start", t.startCommand)
	bot.Handle("/subscribe", t.subscribeCommand)
	bot.Handle("/unsubscribe", t.unsubscribeCommand)
	bot.Handle("/add", t.addSiteCommand)
	bot.Handle("/delete", t.deleteSiteCommand)
	bot.Handle("/list", t.listCommand)

	return t, nil
}

func (t *TGBot) Start() error {
	results, err := t.broker.ConsumeResults()
	if err != nil {
		return fmt.Errorf("failed to register a consumer for checks: %w", err)
	}

	var wg sync.WaitGroup
	done := make(chan struct{})
	exit := make(chan os.Signal, 1)
	signal.Notify(exit, os.Interrupt, syscall.SIGTERM)

	wg.Add(1)
	go func() {
		defer wg.Done()
		t.handleResults(done, results)
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		t.bot.Start()
	}()

	s := <-exit
	slog.Info("exit signal was received", slog.String("signal", s.String()))
	close(done)
	t.bot.Stop()
	wg.Wait()

	return nil
}

func (t *TGBot) handleResults(done <-chan struct{}, results <-chan amqp.Delivery) {
	for {
		select {
		case <-done:
			return
		case msg, ok := <-results:
			if !ok {
				return
			}
			var result model.CheckResult
			err := json.Unmarshal(msg.Body, &result)
			if err != nil {
				slog.Error("failed to parse check result", logger.Error(err))
				return
			}
			t.Notify(result)
		}
	}
}

func (t *TGBot) Notify(result model.CheckResult) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	lastResults, err := t.results.GetLastTwoResultsForSite(ctx, result.Site)
	cancel()
	if err != nil {
		return fmt.Errorf("failed to get last two results for site: %w", err)
	}

	if len(lastResults) == 0 {
		return nil
	}

	var message string
	if len(lastResults) == 1 {
		if !result.IsSuccessful() && !lastResults[0].IsSuccessful() {
			message = fmt.Sprintf(
				"Bad news. The website %s is temporarily unavailable.",
				result.Site.Url,
			)
		}
	} else if result.IsSuccessful() &&
		!lastResults[0].IsSuccessful() &&
		!lastResults[1].IsSuccessful() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		lastSuccessfulResult, err := t.results.GetLastSuccessfulResultForSite(ctx, result.Site)
		cancel()
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("failed to get last successful result for site: %w", err)
		}

		if err == nil {
			message = fmt.Sprintf(
				"Good news! The website %s is back up after %d minutes.",
				result.Site.Url,
				int(time.Since(lastSuccessfulResult.Time).Minutes()),
			)
		} else {
			message = fmt.Sprintf("Good news! The website %s is back up.", result.Site.Url)
		}
	} else if !result.IsSuccessful() &&
		!lastResults[0].IsSuccessful() &&
		lastResults[1].IsSuccessful() {
		message = fmt.Sprintf(
			"Bad news. The website %s is temporarily unavailable.",
			result.Site.Url,
		)
	}

	if message != "" {
		return t.notifySubscribers(result.Site.Url, message)
	}
	return nil
}

func (t *TGBot) notifySubscribers(url string, message string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	chats, err := t.chats.GetAllSubscribedOnSiteChats(ctx, url)
	cancel()
	if err != nil {
		return err
	}

	for _, c := range chats {
		slog.Info("notify subscriber", slog.Int64("chat_id", c.Id), slog.String("message", message))
		if _, err = t.bot.Send(telebot.ChatID(c.Id), message); err != nil {
			slog.Error(
				"failed to send message to chat",
				slog.Int64("chat_id", c.Id),
				slog.String("message", message),
				logger.Error(err),
			)
			return nil
		}
	}

	return nil
}

func (t *TGBot) startCommand(c telebot.Context) error {
	slog.Info("start command", slog.Int64("chat_id", c.Chat().ID))
	return c.Send(`Commands:
	/subscribe - subscribe to updates
	/unsubscribe - unsubscribe from updates
	/add [url] - start monitoring [url] site
	/delete [url] - stop monitoring [url] site
	/list - get all monitored sites
	`)
}

func (t *TGBot) subscribeCommand(c telebot.Context) error {
	slog.Info("subscribe command", slog.Int64("chat_id", c.Chat().ID))

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := t.chats.AddChat(ctx, model.Chat{Id: c.Chat().ID}); err != nil {
		slog.Error(
			"failed to add chat",
			slog.String("command", "subscribe"),
			logger.Error(err),
		)
		return nil
	}

	return c.Send("Successful!")
}

func (t *TGBot) unsubscribeCommand(c telebot.Context) error {
	slog.Info("unsubscribe command", slog.Int64("chat_id", c.Chat().ID))

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := t.chats.UpdateChat(ctx, c.Chat().ID, false); err != nil {
		slog.Error(
			"failed to update chat",
			slog.String("command", "unsubscribe"),
			logger.Error(err),
		)
		return nil
	}

	return c.Send("Successful!")
}

func (t *TGBot) addSiteCommand(c telebot.Context) error {
	chatId := c.Chat().ID
	url := c.Message().Payload

	slog.Info("add site command", slog.Int64("chat_id", chatId), slog.String("url", url))

	url, err := urlpkg.ConvertToExpectedUrl(url)
	if err != nil {
		slog.Error(
			"failed to convert url to expected",
			logger.Error(err),
			slog.String("url", url),
		)
		return c.Reply("Invalid URL!")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := t.sites.AddSiteFromChat(ctx, chatId, url); err != nil {
		slog.Error(
			"failed to add site",
			slog.String("command", "add site"),
			logger.Error(err),
		)
		return nil
	}

	return c.Send("Successful!")
}

func (t *TGBot) deleteSiteCommand(c telebot.Context) error {
	chatId := c.Chat().ID
	url := c.Message().Payload

	slog.Info("delete site command", slog.Int64("chat_id", chatId), slog.String("url", url))

	url, err := urlpkg.ConvertToExpectedUrl(url)
	if err != nil {
		slog.Error(
			"failed to convert url to expected",
			logger.Error(err),
			slog.String("url", url),
		)
		return c.Reply("Invalid URL!")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := t.sites.DeleteSiteFromChat(ctx, chatId, url); err != nil {
		slog.Error(
			"failed to delete site",
			slog.String("command", "delete site"),
			logger.Error(err),
		)
		return nil
	}

	return c.Send("Successful!")
}

func (t *TGBot) listCommand(c telebot.Context) error {
	slog.Info("list command", slog.Int64("chat_id", c.Chat().ID))

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	sites, err := t.sites.GetAllSitesByChatId(ctx, c.Chat().ID)
	if err != nil {
		slog.Error(
			"failed to get all sites by chat id",
			slog.String("command", "list"),
			logger.Error(err),
		)
		return nil
	}

	var b strings.Builder
	for i, site := range sites {
		fmt.Fprintf(&b, "%d) %s\n", i+1, site.Url)
	}

	result := b.String()
	if len(result) == 0 {
		return c.Send("You are not subscribed to any site")
	}
	return c.Send(result)
}
