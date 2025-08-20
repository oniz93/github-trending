package messaging

import (
	"fmt"
	"github.com/streadway/amqp"
)

// MQConnection defines the interface for RabbitMQ connection operations.
type MQConnection interface {
	Consume(queueName string) (<-chan amqp.Delivery, error)
	Publish(queueName string, body []byte) error
	Close() error
	NotifyClose(chan *amqp.Error) chan *amqp.Error
}

// Connection represents a connection to RabbitMQ.
type Connection struct {
	*amqp.Connection
}

// NewConnection establishes a new connection to RabbitMQ.
func NewConnection(url string) (*Connection, error) {
	conn, err := amqp.Dial(url)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to RabbitMQ: %w", err)
	}
	return &Connection{conn}, nil
}

// Consume starts consuming messages from a specified queue.
func (c *Connection) Consume(queueName string) (<-chan amqp.Delivery, error) {
	ch, err := c.Channel()
	if err != nil {
		return nil, fmt.Errorf("failed to open a channel: %w", err)
	}

	_, err = ch.QueueDeclare(
		queueName, // name
		true,      // durable
		false,     // delete when unused
		false,     // exclusive
		false,     // no-wait
		nil,       // arguments
	)
	if err != nil {
		return nil, fmt.Errorf("failed to declare a queue: %w", err)
	}

	msgs, err := ch.Consume(
		queueName, // queue
		"",        // consumer
		false,     // auto-ack
		false,     // exclusive
		false,     // no-local
		false,     // no-wait
		nil,       // args
	)
	if err != nil {
		return nil, fmt.Errorf("failed to register a consumer: %w", err)
	}

	return msgs, nil
}

// Publish sends a message to a specified queue.
func (c *Connection) Publish(queueName string, body []byte) error {
	ch, err := c.Channel()
	if err != nil {
		return fmt.Errorf("failed to open a channel: %w", err)
	}
	defer ch.Close()

	q, err := ch.QueueDeclare(
		queueName, // name
		true,      // durable
		false,     // delete when unused
		false,     // exclusive
		false,     // no-wait
		nil,       // arguments
	)
	if err != nil {
		return fmt.Errorf("failed to declare a queue: %w", err)
	}

	err = ch.Publish(
		"",        // exchange
		q.Name,    // routing key
		false,     // mandatory
		false,     // immediate
		amqp.Publishing{
			ContentType: "application/json",
			Body:        body,
		})
	if err != nil {
		return fmt.Errorf("failed to publish a message: %w", err)
	}
	return nil
}

func (c *Connection) GetQueueMessageCount(queueName string) (int, error) {
	ch, err := c.Channel()
	if err != nil {
		return 0, fmt.Errorf("failed to open a channel: %w", err)
	}
	defer ch.Close()

	q, err := ch.QueueDeclare(
		queueName, // name
		true,      // durable
		false,     // delete when unused
		false,     // exclusive
		false,     // no-wait
		nil,       // arguments
	)
	if err != nil {
		return 0, fmt.Errorf("failed to declare a queue: %w", err)
	}

	return q.Messages, nil
}
