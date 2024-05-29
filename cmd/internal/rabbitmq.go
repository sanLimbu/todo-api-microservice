package internal

import (
	"fmt"

	envvar "github.com/sanLimbu/todo-api/internal/envar"
	"github.com/streadway/amqp"
)

//RabbitMQ.....
type RabbitMQ struct {
	Connection *amqp.Connection
	Channel    *amqp.Channel
}

//NewRabbitMQ instantiates the RabbitMQ instances using configuration defined in environment variables.
func NewRabbitMQ(conf *envvar.Configuration) (*RabbitMQ, error) {
	url, err := conf.Get("RABBITMQ_URL")

	if err != nil {
		return nil, fmt.Errorf("conf.get %w", err)
	}

	conn, err := amqp.Dial(url)
	if err != nil {
		return nil, fmt.Errorf("amqp.dial %w", err)
	}

	ch, err := conn.Channel()
	if err != nil {
		return nil, fmt.Errorf("conn.Channel %w", err)
	}

	err = ch.ExchangeDeclare(
		"tasks", //name
		"topic", //type
		true,    //durable
		false,   //auto-delete
		false,   //internal
		false,   //noWait
		nil,     //arguments
	)
	if err != nil {
		return nil, fmt.Errorf("ch.ExchangeDeclae %w", err)
	}

	if err := ch.Qos(
		1,     //prefetch Count
		0,     //prefetch Size
		false, //global
	); err != nil {
		return nil, fmt.Errorf("ch.Qos %w", err)
	}

	return &RabbitMQ{
		Connection: conn,
		Channel:    ch,
	}, nil

}

// Close ...
func (r *RabbitMQ) Close() {
	r.Connection.Close()
}
