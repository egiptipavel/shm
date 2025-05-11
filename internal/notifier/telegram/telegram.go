package telegram

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
	urlpkg "shm/internal/lib/url"
	"shm/internal/model"
	"shm/internal/service"
	"strings"
	"syscall"
	"time"

	"golang.org/x/sync/errgroup"

	"gopkg.in/telebot.v4"
)

type TGBot struct {
	bot    *telebot.Bot
	broker broker.MessageBroker
	chats  *service.ChatsService
	sites  *service.SitesService
	config config.TelegramBotConfig
}

func New(
	broker broker.MessageBroker,
	chats *service.ChatsService,
	sites *service.SitesService,
	config config.TelegramBotConfig,
) (*TGBot, error) {
	bot, err := telebot.NewBot(telebot.Settings{
		Token:  config.Token,
		Poller: &telebot.LongPoller{Timeout: 10 * time.Second},
	})
	if err != nil {
		return nil, err
	}

	t := &TGBot{
		bot:    bot,
		broker: broker,
		chats:  chats,
		sites:  sites,
		config: config,
	}

	bot.Handle("/start", t.startCommand)
	bot.Handle("/subscribe", t.subscribeCommand)
	bot.Handle("/unsubscribe", t.unsubscribeCommand)
	bot.Handle("/add", t.addSiteCommand)
	bot.Handle("/delete", t.deleteSiteCommand)
	bot.Handle("/list", t.listCommand)

	return t, nil
}

func (t *TGBot) Start() {
	ctx := context.Background()
	ctx, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer stop()
	g, ctx := errgroup.WithContext(ctx)

	notifications, err := t.broker.ConsumeNotifications(ctx)
	if err != nil {
		slog.Error("failed to register a consumer for notifications", sl.Error(err))
		return
	}

	g.Go(func() error {
		return t.handleNotifications(ctx, notifications)
	})

	g.Go(func() error {
		t.bot.Start()
		return nil
	})

	g.Go(func() error {
		<-ctx.Done()
		t.bot.Stop()
		return nil
	})

	if err := g.Wait(); err != nil && !errors.Is(err, context.Canceled) {
		slog.Error("error from telegram bot", sl.Error(err))
	}
}

func (t *TGBot) handleNotifications(
	ctx context.Context,
	notifications <-chan model.Notification,
) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case notification, ok := <-notifications:
			if !ok {
				return fmt.Errorf("queue with notitications was closed")
			}
			if err := t.Notify(ctx, notification); err != nil {
				return fmt.Errorf("failed to handle notification: %w", err)
			}
		}
	}
}

func (t *TGBot) Notify(ctx context.Context, notification model.Notification) error {
	chats, err := t.chats.GetAllSubscribedOnSiteChats(ctx, notification.Url)
	if err != nil {
		return err
	}

	for _, c := range chats {
		slog.Info(
			"sending notification to subscriber",
			slog.Int64("chat_id", c.Id),
			slog.String("message", notification.Message),
		)
		if _, err = t.bot.Send(telebot.ChatID(c.Id), notification.Message); err != nil {
			return fmt.Errorf("failed to send message to chat: %w", err)
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

	if err := t.chats.SubscribeChat(context.Background(), c.Chat().ID); err != nil {
		slog.Error("failed to add chat", slog.String("command", "subscribe"), sl.Error(err))
		return nil
	}

	return c.Send("Successful!")
}

func (t *TGBot) unsubscribeCommand(c telebot.Context) error {
	slog.Info("unsubscribe command", slog.Int64("chat_id", c.Chat().ID))

	if err := t.chats.UnsubscribeChat(context.Background(), c.Chat().ID); err != nil {
		slog.Error("failed to unsubscribe chat", slog.String("command", "unsubscribe"), sl.Error(err))
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

	if err := t.sites.AddSiteFromChat(context.Background(), chatId, url); err != nil {
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

	if err := t.sites.DeleteSiteFromChat(context.Background(), chatId, url); err != nil {
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

	sites, err := t.sites.GetAllSitesByChatId(context.Background(), c.Chat().ID)
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
