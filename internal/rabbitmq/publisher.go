package rabbitmq

import (
	"github.com/sirupsen/logrus"
	"github.com/streadway/amqp"
)

func PublishMessage(command string, routingKey string, commandID string) error {
	conn, err := amqp.Dial("amqp://guest:guest@localhost:5672/")
	if err != nil {
		logrus.Error("Failed to connect to RabbitMQ: ", err)
		return err
	}
	defer conn.Close()

	ch, err := conn.Channel()
	if err != nil {
		logrus.Error("Failed to open a channel", err)
		return err
	}
	defer ch.Close()

	q, err := ch.QueueDeclare(
		routingKey, // name
		true,       // durable: queue definition will survive a broker (RabbitMQ server) restart(persistence(msgs)).
		false,      // delete when unused: queue will be deleted when the last consumer unsubscribes.
		false,      // exclusive: the queue can only be used by the connection that declared it and will be deleted when the connection closes.
		false,      // no-wait: rabbitmq server processes the queue declaration request, but it does not send a confirmation back to the client.
		nil,        // arguments: advanced configurations
	)
	if err != nil {
		logrus.Error("Failed to declare a queue", err)
		return err
	}
	msgBody, err := json.Marshal(CommandMessage{Command: command, CommandID: commandID, RoutingKey: routingKey})
    if err != nil {
        return err
    }
	err = ch.Publish(
		"",     // exchange: ""-> Messages are routed directly to the queue whose name matches the routing key.
		q.Name, // routing key
		fa,   // mandatory:  the server must return a message to the producer if it cannot route the message to a queue. ?=????false
		false,  // immediate: If false, the server will queue the message even if there are no consumers.
		amqp.Publishing{
			ContentType: "text/plain",
			Body:        []byte(command),
		})
	if err != nil {
		logrus.Error("Failed to publish a message", err)
		return err
	}
	logrus.Info("Sent message: ", command, " to queue: ", routingKey)
	return nil
}

/*
 * Channel: Represents a virtual connection within a physical connection to RabbitMQ. Used to perform various operations but does not have a name.
 * Queue: Named entities where messages are sent to and received from. The name is specified during queue declaration, not channel creation.
 */
