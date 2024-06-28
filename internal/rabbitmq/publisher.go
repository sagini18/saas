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

func Start(connection *types.Connection) {
	SetConnection(connection)
}

func SetConnection(connection *types.Connection) {
	_, err := connection.Channel.QueueDeclare(
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

	go HandleReconnect(connection)
}

func HandleReconnect(connection *types.Connection) {
	for {
		connErr := <-connection.Conn.NotifyClose(make(chan *amqp.Error))
		log.Print(connErr)
		time.Sleep(5 * time.Second)

		// Recreate connection and channel
		newConn, err := types.NewConnection()
		if err != nil {
			log.Fatalf("Failed to reconnect: %v", err)
		}
		connection.Conn = newConn.Conn
		connection.Channel = newConn.Channel
		SetConnection(connection)
	}
}

func PublishMessage(command string, routingKey string, commandID string, channel *amqp.Channel) error {
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

func ConsumeResponse(routingKey string, commandID string, channel *amqp.Channel) ([]byte, error) {
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
