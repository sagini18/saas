package main

import (
	"os"

	"github.com/sagini18/saas/internal/rabbitmq"
	"github.com/sagini18/saas/internal/server"
	"github.com/sagini18/saas/internal/types"
	"github.com/sirupsen/logrus"
)

func main() {
	configureLogger()

	connection, err := types.NewConnection()
	if err != nil {
		logrus.Fatalf("Failed to create connection: %v", err)
	}
	defer connection.Conn.Close()
	defer connection.Channel.Close()

	rabbitmq.Start(connection)
	server.Start(connection.Channel)
}

func configureLogger() {
	logrus.SetFormatter(&logrus.JSONFormatter{})
	logrus.SetOutput(os.Stdout)
	logrus.SetLevel(logrus.DebugLevel)
}
