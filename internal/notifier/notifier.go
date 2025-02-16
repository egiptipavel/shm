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
	bot     *telebot.Bot
	mu      sync.RWMutex
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

		err := t.storage.AddSubscriber(ctx, storage.Subscriber{ChatID: c.Chat().ID})
		if err != nil {
			return err
		}

		log.Printf("new subscriber: %d", c.Chat().ID)
		return c.Send("Successful!")
	})
	bot.Handle("/unsubscribe", func(c telebot.Context) error {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		err := t.storage.DeleteSubscriber(ctx, c.Chat().ID)
		if err != nil {
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
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	subscribers, err := t.storage.GetAllSubscribers(ctx)
	cancel()
	if err != nil {
		return err
	}

	message := fmt.Sprintf("successful checking of %s site: code %d, latency %dms", result.Url, result.Code, result.Latency)
	for _, s := range subscribers {
		t.bot.Send(telebot.ChatID(s.ChatID), message)
	}

	return nil
}
