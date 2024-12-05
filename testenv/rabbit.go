package testenv

import (
	"log/slog"
	"os"
	"time"

	"github.com/streadway/amqp"
)

type rabbit struct {
	conn    *amqp.Connection
	channel *amqp.Channel
}

func (r *rabbit) connect(url string) {
	conn, err := amqp.Dial(url)
	if err != nil {
		slog.Error("Failed to connect to RabbitMQ", "error", err)
		os.Exit(1)
	}
	r.conn = conn

	ch, err := conn.Channel()
	if err != nil {
		slog.Error("Failed to open a channel", "error", err)
		os.Exit(1)
	}
	r.channel = ch
}

func (r *rabbit) DeclareQueue(name string, durable bool) {
	_, err := r.channel.QueueDeclare(
		name,    // name
		durable, // durable
		false,   // delete when unused
		false,   // exclusive
		false,   // no-wait
		nil,     // arguments
	)
	if err != nil {
		slog.Error("Failed to declare a queue", "error", err)
		os.Exit(1)
	}
}

func (r *rabbit) SendMessageToQ(body string, routingKey string, timestamp *time.Time) {
	pub := amqp.Publishing{
		ContentType: "text/plain",
		Body:        []byte(body),
	}
	if timestamp != nil {
		pub.Timestamp = *timestamp
	}
	err := r.channel.Publish(
		"",         // exchange
		routingKey, // routing key
		false,      // mandatory
		false,      // immediate
		pub)
	if err != nil {
		slog.Error("Failed to publish a message", "body", body, "error", err)
	}
}
