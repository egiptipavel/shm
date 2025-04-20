package telegram

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"shm/internal/broker/rabbitmq"
	"shm/internal/lib/sl"
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
	bot    *telebot.Bot
	broker *rabbitmq.RabbitMQ
	chats  *repository.Chats
	sites  *repository.Sites
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
		bot:    bot,
		broker: broker,
		chats:  repository.NewChatsRepo(db),
		sites:  repository.NewSitesRepo(db),
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
	notifications, err := t.broker.ConsumeNotifications()
	if err != nil {
		return fmt.Errorf("failed to register a consumer for notifications: %w", err)
	}

	var wg sync.WaitGroup
	done := make(chan struct{})
	exit := make(chan os.Signal, 1)
	signal.Notify(exit, os.Interrupt, syscall.SIGTERM)

	wg.Add(1)
	go func() {
		defer wg.Done()
		t.handleNotifications(done, notifications)
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

func (t *TGBot) handleNotifications(done <-chan struct{}, notifications <-chan amqp.Delivery) {
	for {
		select {
		case <-done:
			return
		case msg, ok := <-notifications:
			if !ok {
				return
			}
			var notification model.Notification
			err := json.Unmarshal(msg.Body, &notification)
			if err != nil {
				slog.Error("failed to parse notification", sl.Error(err))
				return
			}
			t.Notify(notification)
		}
	}
}

func (t *TGBot) Notify(notification model.Notification) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	chats, err := t.chats.GetAllSubscribedOnSiteChats(ctx, notification.Url)
	cancel()
	if err != nil {
		return err
	}

	for _, c := range chats {
		slog.Info(
			"notify subscriber",
			slog.Int64("chat_id", c.Id),
			slog.String("message", notification.Message),
		)
		if _, err = t.bot.Send(telebot.ChatID(c.Id), notification.Message); err != nil {
			slog.Error(
				"failed to send message to chat",
				slog.Int64("chat_id", c.Id),
				slog.String("message", notification.Message),
				sl.Error(err),
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
			sl.Error(err),
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
			sl.Error(err),
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
			sl.Error(err),
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
			sl.Error(err),
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
			sl.Error(err),
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
			sl.Error(err),
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
			sl.Error(err),
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
