package server

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/sagini18/saas/internal/rabbitmq"
	"github.com/sagini18/saas/internal/types"
	"github.com/sirupsen/logrus"
	"github.com/streadway/amqp"
)

func commandHandler(w http.ResponseWriter, r *http.Request, channel *amqp.Channel) {
	var cmdMsg types.CommandMessage

	err := json.NewDecoder(r.Body).Decode(&cmdMsg)
	if err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	logrus.Infof("Received command: %s with routing key: %s", cmdMsg.Command, cmdMsg.RoutingKey)

	err = rabbitmq.PublishMessage(cmdMsg.Command, cmdMsg.RoutingKey, cmdMsg.CommandID, channel)
	if err != nil {
		http.Error(w, "Failed to publish the message: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Create a channel to receive the response
	responseChan := make(chan []byte)
	errorChan := make(chan error)

	go func() {
		res, err := rabbitmq.ConsumeResponse(cmdMsg.RoutingKey, cmdMsg.CommandID, channel)
		if err != nil {
			errorChan <- err
			return
		}
		responseChan <- res
	}()

	select {
	case res := <-responseChan:
		if res == nil {
			w.WriteHeader(http.StatusProcessing)
			w.Write([]byte("Message received and queued successfully"))
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(res))
	case err := <-errorChan:
		http.Error(w, "Failed to consume the response: "+err.Error(), http.StatusInternalServerError)
	case <-time.After(5 * time.Second):
		w.WriteHeader(http.StatusProcessing)
		w.Write([]byte("Message received and queued successfully"))
	}
}

func Start(channel *amqp.Channel) {
	http.HandleFunc("/api/v1/commands", func(w http.ResponseWriter, r *http.Request) {
		commandHandler(w, r, channel)
	})
	logrus.Info("Starting server on port 5050...")
	if err := http.ListenAndServe(":5050", nil); err != nil {
		logrus.Fatalf("Server failed: %s", err)
	}
}
