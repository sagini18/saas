package main

import (
	"os"

	"github.com/sagini18/saas/internal/server"
	"github.com/sirupsen/logrus"
)

func main() {
	configureLogger()

	server.Start()
}

func configureLogger() {
	logrus.SetFormatter(&logrus.JSONFormatter{})
	logrus.SetOutput(os.Stdout)
	logrus.SetLevel(logrus.DebugLevel)
}
