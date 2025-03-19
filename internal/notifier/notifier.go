package notifier

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"shm/internal/storage"
	urlpkg "shm/internal/url"
	"strings"
	"time"

	"gopkg.in/telebot.v4"
)

type Notifier interface {
	Notify(result storage.CheckResult) error
}

type TGBot struct {
	bot     *telebot.Bot
	storage *storage.Storage
}

func NewTGBot(token string, stor *storage.Storage) (*TGBot, error) {
	bot, err := telebot.NewBot(telebot.Settings{
		Token:  token,
		Poller: &telebot.LongPoller{Timeout: 10 * time.Second},
	})
	if err != nil {
		return nil, err
	}

	t := &TGBot{
		bot:     bot,
		storage: stor,
	}

	bot.Handle("/start", t.startCommand())
	bot.Handle("/subscribe", t.subscribeCommand())
	bot.Handle("/unsubscribe", t.unsubscribeCommand())
	bot.Handle("/add", t.addSiteCommand())
	bot.Handle("/delete", t.deleteSiteCommand())
	bot.Handle("/list", t.listCommand())

	return t, nil
}

func (t *TGBot) Start() {
	t.bot.Start()
}

func (t *TGBot) Notify(result storage.CheckResult) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	lastResult, err := t.storage.GetLastResultForSite(ctx, result.Site)
	cancel()

	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return err
	}

	var message string
	if err == nil {
		if (!lastResult.Code.Valid || lastResult.Code.Int64 != 200) &&
			result.Code.Valid && result.Code.Int64 == 200 {
			message = fmt.Sprintf(
				"Good news! The website %s is back up after %d minutes.",
				result.Site.Url,
				int(time.Since(lastResult.Time).Minutes()),
			)
		} else if lastResult.Code.Valid && lastResult.Code.Int64 == 200 &&
			(!result.Code.Valid || result.Code.Int64 != 200) {
			message = fmt.Sprintf(
				"Bad news. The website %s is temporarily unavailable.",
				result.Site.Url,
			)
		}
	} else {
		if !result.Code.Valid || result.Code.Int64 != 200 {
			message = fmt.Sprintf(
				"Bad news. The website %s is temporarily unavailable.",
				result.Site.Url,
			)
		}
	}

	if message != "" {
		return t.notifySubscribers(result.Site.Url, message)
	}
	return nil
}

func (t *TGBot) notifySubscribers(url string, message string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	chats, err := t.storage.GetAllSubscribedOnSiteChats(ctx, url)
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
				slog.String("error", err.Error()),
			)
			return nil
		}
	}

	return nil
}

func (t *TGBot) startCommand() func(c telebot.Context) error {
	return func(c telebot.Context) error {
		slog.Info("start command", slog.Int64("chat_id", c.Chat().ID))
		return c.Send(`Commands:
	/subscribe - subscribe to updates
	/unsubscribe - unsubscribe from updates
	/add [url] - start monitoring [url] site
	/delete [url] - stop monitoring [url] site
	/list - get all monitored sites
	`)
	}
}

func (t *TGBot) subscribeCommand() func(c telebot.Context) error {
	return func(c telebot.Context) error {
		slog.Info("subscribe command", slog.Int64("chat_id", c.Chat().ID))

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := t.storage.AddChat(ctx, storage.Chat{Id: c.Chat().ID}); err != nil {
			slog.Error(
				"failed to add chat",
				slog.String("command", "subscribe"),
				slog.String("error", err.Error()),
			)
			return nil
		}

		return c.Send("Successful!")
	}
}

func (t *TGBot) unsubscribeCommand() func(c telebot.Context) error {
	return func(c telebot.Context) error {
		slog.Info("unsubscribe command", slog.Int64("chat_id", c.Chat().ID))

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := t.storage.UpdateChat(ctx, c.Chat().ID, false); err != nil {
			slog.Error(
				"failed to update chat",
				slog.String("command", "unsubscribe"),
				slog.String("error", err.Error()),
			)
			return nil
		}

		return c.Send("Successful!")
	}
}

func (t *TGBot) addSiteCommand() func(c telebot.Context) error {
	return func(c telebot.Context) error {
		chatId := c.Chat().ID
		url := c.Message().Payload

		slog.Info("add site command", slog.Int64("chat_id", chatId), slog.String("url", url))

		url, err := urlpkg.ConvertToExpectedUrl(url)
		if err != nil {
			slog.Error(
				"failed to convert url to expected",
				slog.String("error", err.Error()),
				slog.String("url", url),
			)
			return c.Reply("Invalid URL!")
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := t.storage.AddSite(ctx, chatId, url); err != nil {
			slog.Error(
				"failed to add site",
				slog.String("command", "add site"),
				slog.String("error", err.Error()),
			)
			return nil
		}

		return c.Send("Successful!")
	}
}

func (t *TGBot) deleteSiteCommand() func(c telebot.Context) error {
	return func(c telebot.Context) error {
		chatId := c.Chat().ID
		url := c.Message().Payload

		slog.Info("delete site command", slog.Int64("chat_id", chatId), slog.String("url", url))

		url, err := urlpkg.ConvertToExpectedUrl(url)
		if err != nil {
			slog.Error(
				"failed to convert url to expected",
				slog.String("error", err.Error()),
				slog.String("url", url),
			)
			return c.Reply("Invalid URL!")
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := t.storage.DeleteSite(ctx, chatId, url); err != nil {
			slog.Error(
				"failed to delete site",
				slog.String("command", "delete site"),
				slog.String("error", err.Error()),
			)
			return nil
		}

		return c.Send("Successful!")
	}
}

func (t *TGBot) listCommand() func(c telebot.Context) error {
	return func(c telebot.Context) error {
		slog.Info("list command", slog.Int64("chat_id", c.Chat().ID))

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		sites, err := t.storage.GetAllSitesByChatId(ctx, c.Chat().ID)
		if err != nil {
			slog.Error(
				"failed to get all sites by chat id",
				slog.String("command", "list"),
				slog.String("error", err.Error()),
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
}
