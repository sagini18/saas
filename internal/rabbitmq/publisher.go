package rabbitmq

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/sagini18/saas/internal/types"
	"github.com/sirupsen/logrus"
	"github.com/streadway/amqp"
)

var (
	conn    *amqp.Connection
	channel *amqp.Channel
)

func Start() {
	connectToRabbitMQ()
	defer conn.Close()
	defer channel.Close()
}

func connectToRabbitMQ() {
	var err error
	conn, err = amqp.Dial("amqp://guest:guest@localhost:5672/")
	if err != nil {
		log.Fatalf("Failed to connect to RabbitMQ: %v", err)
	}

	channel, err = conn.Channel()
	if err != nil {
		log.Fatalf("Failed to open a channel: %v", err)
	}

	_, err = channel.QueueDeclare(
		"k8s-1-command", // name
		true,            // durable
		false,           // auto-deleted
		false,           // internal
		false,           // no-wait
		nil,             // arguments
	)
	if err != nil {
		log.Fatalf("Failed to declare an exchange: %v", err)
	}

	go HandleReconnect()
}

func HandleReconnect() {
	for {
		connErr := <-conn.NotifyClose(make(chan *amqp.Error))
		log.Print(connErr)
		time.Sleep(5 * time.Second)
		connectToRabbitMQ()
	}
}

func PublishMessage(command string, routingKey string, commandID string) error {
	msgBody, err := json.Marshal(types.CommandMessage{Command: command, CommandID: commandID, RoutingKey: routingKey})
	if err != nil {
		return err
	}
	err = channel.Publish(
		"",         // exchange: ""-> Messages are routed directly to the queue whose name matches the routing key.
		routingKey, // routing key
		true,       // mandatory:  the server must return a message to the producer if it cannot route the message to a queue. ?=????false
		false,      // immediate: If false, the server will queue the message even if there are no consumers.
		amqp.Publishing{
			ContentType:  "application/json",
			Body:         msgBody,
			DeliveryMode: amqp.Persistent, // make message persistent
		})
	if err != nil {
		logrus.Error("Failed to publish a message", err)
		return err
	}
	logrus.Info("Sent message: ", command, " to queue: ", routingKey)
	return nil
}

func ConsumeResponse(routingKey string, commandID string) ([]byte, error) {
	msgs, err := channel.Consume(
		routingKey+"-response", // queue
		"",                     // consumer
		true,                   // auto-ack
		false,                  // exclusive
		false,                  // no-local
		false,                  // no-wait
		nil,                    // args
	)
	if err != nil {
		logrus.Error("Failed to consume messages: ", err)
		return nil, err
	}
	for msg := range msgs {
		fmt.Println("Received Back: ", string(msg.Body))
		var decodedMsg types.CommandResponse
		if err := json.NewDecoder(bytes.NewReader(msg.Body)).Decode(&decodedMsg); err != nil {
			logrus.Error("Failed to decode message: ", err)
			return nil, err
		}
		if decodedMsg.CommandID == commandID && decodedMsg.RoutingKey == routingKey {
			return msg.Body, nil
		}
	}
	return nil, nil
}

// func PublishMessage(command string, routingKey string, commandID string) error {
// 	conn, err := amqp.Dial("amqp://guest:guest@localhost:5672/")
// 	if err != nil {
// 		logrus.Error("Failed to connect to RabbitMQ: ", err)
// 		return err
// 	}
// 	defer conn.Close()

// 	ch, err := conn.Channel()
// 	if err != nil {
// 		logrus.Error("Failed to open a channel", err)
// 		return err
// 	}
// 	defer ch.Close()

// 	q, err := ch.QueueDeclare(
// 		routingKey, // name
// 		true,       // durable: queue definition will survive a broker (RabbitMQ server) restart(persistence(msgs)).
// 		false,      // delete when unused: queue will be deleted when the last consumer unsubscribes.
// 		false,      // exclusive: the queue can only be used by the connection that declared it and will be deleted when the connection closes.
// 		false,      // no-wait: rabbitmq server processes the queue declaration request, but it does not send a confirmation back to the client.
// 		nil,        // arguments: advanced configurations
// 	)
// 	if err != nil {
// 		logrus.Error("Failed to declare a queue", err)
// 		return err
// 	}
// 	msgBody, err := json.Marshal(types.CommandMessage{Command: command, CommandID: commandID, RoutingKey: routingKey})
// 	if err != nil {
// 		return err
// 	}
// 	err = ch.Publish(
// 		"",     // exchange: ""-> Messages are routed directly to the queue whose name matches the routing key.
// 		q.Name, // routing key
// 		true,   // mandatory:  the server must return a message to the producer if it cannot route the message to a queue. ?=????false
// 		false,  // immediate: If false, the server will queue the message even if there are no consumers.
// 		amqp.Publishing{
// 			ContentType:  "application/json",
// 			Body:         msgBody,
// 			DeliveryMode: amqp.Persistent, // make message persistent
// 		})
// 	if err != nil {
// 		logrus.Error("Failed to publish a message", err)
// 		return err
// 	}
// 	logrus.Info("Sent message: ", command, " to queue: ", routingKey)
// 	return nil
// }

/*
 * Channel: Represents a virtual connection within a physical connection to RabbitMQ. Used to perform various operations but does not have a name.
 * Queue: Named entities where messages are sent to and received from. The name is specified during queue declaration, not channel creation.
 */
