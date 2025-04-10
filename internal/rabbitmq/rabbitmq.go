package rabbitmq

import amqp "github.com/rabbitmq/amqp091-go"

func DeclareChecksQueue(ch *amqp.Channel) (amqp.Queue, error) {
	return ch.QueueDeclare(
		"checks", // name
		false,    // durable
		false,    // delete when unused
		false,    // exclusive
		false,    // no-wait
		nil,      // arguments
	)
}
