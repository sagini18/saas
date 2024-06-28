package types

import "github.com/streadway/amqp"

type Connection struct {
	Conn    *amqp.Connection
	Channel *amqp.Channel
}

func NewConnection() (*Connection, error) {
	conn, err := amqp.Dial("amqp://guest:guest@localhost:5672/")
	if err != nil {
		return nil, err
	}

	channel, err := conn.Channel()
	if err != nil {
		return nil, err
	}

	return &Connection{
		Conn:    conn,
		Channel: channel,
	}, nil
}
