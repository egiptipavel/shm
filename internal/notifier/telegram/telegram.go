package telegram

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"shm/internal/lib/logger"
	urlpkg "shm/internal/lib/url"
	"shm/internal/model"
	"shm/internal/repository"
	"strings"
	"time"

	"gopkg.in/telebot.v4"
)

type TGBot struct {
	bot     *telebot.Bot
	chats   *repository.Chats
	results *repository.Results
	sites   *repository.Sites
}

func New(token string, db *sql.DB) (*TGBot, error) {
	bot, err := telebot.NewBot(telebot.Settings{
		Token:  token,
		Poller: &telebot.LongPoller{Timeout: 10 * time.Second},
	})
	if err != nil {
		return nil, err
	}

	t := &TGBot{
		bot,
		repository.NewChatsRepo(db),
		repository.NewResultsRepo(db),
		repository.NewSitesRepo(db),
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
	t.bot.Start()
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
