package rabbitmq

import (
	"context"
	"fmt"

	amqp "github.com/rabbitmq/amqp091-go"
)

type RabbitMQ struct {
	conn           *amqp.Connection
	ch             *amqp.Channel
	checksQ        amqp.Queue
	resultQ        amqp.Queue
	notificationsQ amqp.Queue
}

func New(url string) (*RabbitMQ, error) {
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

	checksQ, err := declareChecksQueue(ch)
	if err != nil {
		return nil, fmt.Errorf("failed to declare a checks queue: %w", err)
	}

	resultsQ, err := declareResultsQueue(ch)
	if err != nil {
		return nil, fmt.Errorf("failed to declare a results queue: %w", err)
	}

	notificationsQ, err := declareNotificationsQueue(ch)
	if err != nil {
		return nil, fmt.Errorf("failed to declare a notifications queue: %w", err)
	}

	return &RabbitMQ{
		conn:           conn,
		ch:             ch,
		checksQ:        checksQ,
		resultQ:        resultsQ,
		notificationsQ: notificationsQ,
	}, nil
}

func declareChecksQueue(ch *amqp.Channel) (amqp.Queue, error) {
	return ch.QueueDeclare(
		"checks", // name
		false,    // durable
		false,    // delete when unused
		false,    // exclusive
		false,    // no-wait
		nil,      // arguments
	)
}

func declareResultsQueue(ch *amqp.Channel) (amqp.Queue, error) {
	return ch.QueueDeclare(
		"results", // name
		false,     // durable
		false,     // delete when unused
		false,     // exclusive
		false,     // no-wait
		nil,       // arguments
	)
}

func declareNotificationsQueue(ch *amqp.Channel) (amqp.Queue, error) {
	return ch.QueueDeclare(
		"notifications", // name
		false,           // durable
		false,           // delete when unused
		false,           // exclusive
		false,           // no-wait
		nil,             // arguments
	)
}

func (r *RabbitMQ) ConsumeChecks() (<-chan amqp.Delivery, error) {
	return r.ch.Consume(
		r.checksQ.Name, // queue
		"",             // consumer
		true,           // auto-ack
		false,          // exclusive
		false,          // no-local
		false,          // no-wait
		nil,            // args
	)
}

func (r *RabbitMQ) ConsumeResults() (<-chan amqp.Delivery, error) {
	return r.ch.Consume(
		r.resultQ.Name, // queue
		"",             // consumer
		true,           // auto-ack
		false,          // exclusive
		false,          // no-local
		false,          // no-wait
		nil,            // args
	)
}

func (r *RabbitMQ) ConsumeNotifications() (<-chan amqp.Delivery, error) {
	return r.ch.Consume(
		r.notificationsQ.Name, // queue
		"",                    // consumer
		true,                  // auto-ack
		false,                 // exclusive
		false,                 // no-local
		false,                 // no-wait
		nil,                   // args
	)
}

func (r *RabbitMQ) PublishToChecks(ctx context.Context, msg amqp.Publishing) error {
	return r.ch.PublishWithContext(ctx, "", r.checksQ.Name, false, false, msg)
}

func (r *RabbitMQ) PublishToResults(ctx context.Context, msg amqp.Publishing) error {
	return r.ch.PublishWithContext(ctx, "", r.resultQ.Name, false, false, msg)
}

func (r *RabbitMQ) PublishToNotifications(ctx context.Context, msg amqp.Publishing) error {
	return r.ch.PublishWithContext(ctx, "", r.notificationsQ.Name, false, false, msg)
}

func (r *RabbitMQ) Close() {
	defer r.conn.Close()
	defer r.ch.Close()
}
