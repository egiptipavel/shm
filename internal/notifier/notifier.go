package notifier

import (
	"context"
	"fmt"
	"log"
	"shm/internal/storage"
	"sync"
	"time"

	"gopkg.in/telebot.v4"
)

type Notifier interface {
	Notify(result storage.CheckResult) error
}

type TGBot struct {
	mu      sync.RWMutex
	bot     *telebot.Bot
	storage *storage.Storage
	failed  map[string]time.Time
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
		failed:  make(map[string]time.Time),
	}
	bot.Handle("/start", func(c telebot.Context) error {
		return c.Send(`Site Health Monitor Bot
Commands:
/subscribe - subscribe to updates
/unsubscribe - unsubscribe from updates
`)
	})
	bot.Handle("/subscribe", func(c telebot.Context) error {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := t.storage.AddSubscriber(ctx, storage.Subscriber{ChatID: c.Chat().ID}); err != nil {
			return err
		}

		log.Printf("new subscriber: %d", c.Chat().ID)
		return c.Send("Successful!")
	})
	bot.Handle("/unsubscribe", func(c telebot.Context) error {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := t.storage.DeleteSubscriber(ctx, c.Chat().ID); err != nil {
			return err
		}

		log.Printf("unsubscriber: %d", c.Chat().ID)
		return c.Send("Successful!")
	})

	return t, nil
}

func (t *TGBot) Start() {
	t.bot.Start()
}

func (t *TGBot) Notify(result storage.CheckResult) error {
	var message string

	t.mu.Lock()
	if ftime, ok := t.failed[result.Url]; ok && result.Code == 200 {
		delete(t.failed, result.Url)

		message = fmt.Sprintf(
			"Good news! The website %s is back up after %d minutes.",
			result.Url,
			int(time.Since(ftime).Minutes()),
		)
	} else if !ok && result.Code != 200 {
		t.failed[result.Url] = time.Now()

		message = fmt.Sprintf(
			"Bad news. The website %s is temporarily unavailable.",
			result.Url,
		)
	}
	t.mu.Unlock()

	if message != "" {
		return t.notifyAllSubscribers(message)
	}
	return nil
}

func (t *TGBot) notifyAllSubscribers(message string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	subs, err := t.storage.GetAllSubscribers(ctx)
	cancel()
	if err != nil {
		return err
	}

	for _, s := range subs {
		if _, err = t.bot.Send(telebot.ChatID(s.ChatID), message); err != nil {
			return err
		}
	}

	return nil
}
