package broker

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"shm/internal/lib/sl"
	"shm/internal/model"
	"sync"

	amqp "github.com/rabbitmq/amqp091-go"
)

var _ MessageBroker = &RabbitMQ{}

type RabbitMQ struct {
	wg             sync.WaitGroup
	conn           *amqp.Connection
	ch             *amqp.Channel
	sitesQ         amqp.Queue
	resultQ        amqp.Queue
	notificationsQ amqp.Queue
	closed         chan struct{}
}

func NewRabbitMQ(url string) (*RabbitMQ, error) {
	conn, err := amqp.Dial(url)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to RabbitMQ: %w", err)
	}

	ch, err := conn.Channel()
	if err != nil {
		return nil, fmt.Errorf("failed to create channel: %w", err)
	}

	err = ch.Qos(
		1,     // prefetch count
		0,     // prefetch size
		false, // global
	)
	if err != nil {
		return nil, fmt.Errorf("failed to set QoS: %w", err)
	}

	sitesQ, err := declareQueue(ch, "sites")
	if err != nil {
		return nil, fmt.Errorf("failed to declare a sites queue: %w", err)
	}

	resultsQ, err := declareQueue(ch, "results")
	if err != nil {
		return nil, fmt.Errorf("failed to declare a results queue: %w", err)
	}

	notificationsQ, err := declareQueue(ch, "notifications")
	if err != nil {
		return nil, fmt.Errorf("failed to declare a notifications queue: %w", err)
	}

	return &RabbitMQ{
		conn:           conn,
		ch:             ch,
		sitesQ:         sitesQ,
		resultQ:        resultsQ,
		notificationsQ: notificationsQ,
		closed:         make(chan struct{}),
	}, nil
}

func declareQueue(ch *amqp.Channel, name string) (amqp.Queue, error) {
	return ch.QueueDeclare(
		name,  // name
		false, // durable
		false, // delete when unused
		false, // exclusive
		false, // no-wait
		nil,   // arguments
	)
}

func (r *RabbitMQ) ConsumeSites(ctx context.Context) (<-chan model.Site, error) {
	return consumeRoutine[model.Site](r, ctx, r.sitesQ.Name)
}

func (r *RabbitMQ) ConsumeResults(ctx context.Context) (<-chan model.CheckResult, error) {
	return consumeRoutine[model.CheckResult](r, ctx, r.resultQ.Name)
}

func (r *RabbitMQ) ConsumeNotifications(ctx context.Context) (<-chan model.Notification, error) {
	return consumeRoutine[model.Notification](r, ctx, r.notificationsQ.Name)
}

func consumeRoutine[T any](r *RabbitMQ, ctx context.Context, queue string) (<-chan T, error) {
	msgs, err := r.consumeMessages(ctx, queue)
	if err != nil {
		return nil, err
	}

	objects := make(chan T)
	r.wg.Add(1)
	go func() {
		defer r.wg.Done()
		defer close(objects)
		for msg := range msgs {
			var object T
			if err := json.Unmarshal(msg.Body, &object); err != nil {
				slog.Error("failed to parse message body", sl.Error(err))
				continue
			}
			select {
			case <-r.closed:
				return
			case objects <- object:
			}
		}
	}()

	return objects, nil
}

func (r *RabbitMQ) consumeMessages(
	ctx context.Context,
	queue string,
) (<-chan amqp.Delivery, error) {
	return r.ch.ConsumeWithContext(
		ctx,
		queue, // queue
		"",    // consumer
		true,  // auto-ack
		false, // exclusive
		false, // no-local
		false, // no-wait
		nil,   // args
	)
}

func (r *RabbitMQ) PublishSite(_ context.Context, site model.Site) error {
	body, err := json.Marshal(site)
	if err != nil {
		return fmt.Errorf("failed to marshal site: %w", err)
	}

	msg := amqp.Publishing{
		ContentType: "application/json",
		Body:        body,
	}
	return r.ch.Publish("", r.sitesQ.Name, false, false, msg)
}

func (r *RabbitMQ) PublishResult(_ context.Context, result model.CheckResult) error {
	body, err := json.Marshal(result)
	if err != nil {
		return fmt.Errorf("failed to marshal result: %w", err)
	}

	msg := amqp.Publishing{
		ContentType: "application/json",
		Body:        body,
	}
	return r.ch.Publish("", r.resultQ.Name, false, false, msg)
}

func (r *RabbitMQ) PublishNotification(_ context.Context, notification model.Notification) error {
	body, err := json.Marshal(notification)
	if err != nil {
		return fmt.Errorf("failed to marshal notification: %w", err)
	}

	msg := amqp.Publishing{
		ContentType: "application/json",
		Body:        body,
	}
	return r.ch.Publish("", r.notificationsQ.Name, false, false, msg)
}

func (r *RabbitMQ) Close() {
	defer r.conn.Close()
	defer r.ch.Close()

	close(r.closed)
	r.wg.Wait()
}
