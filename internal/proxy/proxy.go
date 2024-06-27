package main

import (
	"bytes"
	"context"
	"encoding/json"
	"log"
	"net"
	"sync"
	"time"

	"github.com/sagini18/saas/internal/types"
	pb "github.com/sagini18/saas/proto"
	"github.com/sirupsen/logrus"
	"github.com/streadway/amqp"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type server struct {
	pb.UnimplementedProxyServer
	subscriptions map[string][]string // map of queue to list of agent IDs
	clients       map[string]pb.Proxy_StreamCommandsServer
	mu            sync.Mutex
}

func validateToken(ctx context.Context) bool {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return false
	}

	token := md["authorization"]
	if len(token) == 0 {
		return false
	}

	// Verify the token (implement your token validation logic here)
	return token[0] == "valid-token"
}

func (s *server) Subscribe(ctx context.Context, req *pb.SubscriptionRequest) (*pb.SubscriptionResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !validateToken(ctx) {
		return nil, status.Errorf(codes.PermissionDenied, "Forbidden")
	}

	logrus.Info("Valid subscription request")
	if _, exists := s.subscriptions[req.Queue]; !exists {
		s.subscriptions[req.Queue] = []string{}
	}
	s.subscriptions[req.Queue] = append(s.subscriptions[req.Queue], req.AgentId)
	return &pb.SubscriptionResponse{Message: "Subscribed successfully"}, nil
}

func (s *server) StreamCommands(stream pb.Proxy_StreamCommandsServer) error {
	// Extract agent ID from the first received message
	initialReq, err := stream.Recv()
	if err != nil {
		return err
	}
	agentID := initialReq.AgentId

	s.mu.Lock()
	s.clients[agentID] = stream
	s.mu.Unlock()

	// Keep the stream open
	for {
		cmd, err := stream.Recv()
		if err != nil {
			return err
		}
		logrus.Infof("All:%s", cmd.Command)
		logrus.Errorf("Received command: %s", cmd.Command.Command)
	}
}

func main() {
	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	logrus.Info("Server started on port 50051")

	s := grpc.NewServer()
	server := &server{
		subscriptions: make(map[string][]string),
		clients:       make(map[string]pb.Proxy_StreamCommandsServer),
	}
	pb.RegisterProxyServer(s, server)

	go server.startConsuming()

	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}

func (s *server) startConsuming() {
	for {
		conn, err := amqp.Dial("amqp://guest:guest@localhost:5672/")
		if err != nil {
			logrus.Errorf("Failed to connect to RabbitMQ: %s. Retrying in 5 seconds...", err)
			time.Sleep(5 * time.Second)
			continue
		}
		logrus.Info("Connected to RabbitMQ")

		s.consume(conn)

		conn.Close()
		logrus.Info("RabbitMQ connection closed. Reconnecting in 5 seconds...")
		time.Sleep(5 * time.Second)
	}
}

func (s *server) consume(conn *amqp.Connection) {
	for {
		ch, err := conn.Channel()
		if err != nil {
			logrus.Errorf("Failed to open a channel: %s. Retrying in 5 seconds...", err)
			time.Sleep(5 * time.Second)
			continue
		}
		defer ch.Close()

		logrus.Info("Opened a channel")

		// // Declare queues and start consuming messages
		go func() {
			for {
				s.mu.Lock()
				for queue := range s.subscriptions {
					go s.consumeFromQueue(ch, queue)
				}
				s.mu.Unlock()
				time.Sleep(5 * time.Second) // Adjust sleep time to control frequency
			}
		}()

		// Monitor connection and channel health
		notifyCloseConn := conn.NotifyClose(make(chan *amqp.Error))
		notifyCloseCh := ch.NotifyClose(make(chan *amqp.Error))
		select {
		case err := <-notifyCloseConn:
			log.Printf("RabbitMQ connection closed: %v. Reconnecting...", err)
			return
		case err := <-notifyCloseCh:
			log.Printf("RabbitMQ channel closed: %v. Reconnecting...", err)
			return
		}
	}
}

func (s *server) consumeFromQueue(ch *amqp.Channel, queue string) {
	_, err := ch.QueueDeclare(
		queue, // name
		true,  // durable
		false, // delete when unused
		false, // exclusive
		false, // no-wait
		nil,   // arguments
	)
	if err != nil {
		logrus.Errorf("Failed to declare a queue %s: %s", queue, err)
		return
	}

	msgs, err := ch.Consume(
		queue, // queue
		"",    // consumer tag
		false, // auto-ack
		false, // exclusive
		false, // no-local
		false, // no-wait
		nil,   // args
	)
	if err != nil {
		logrus.Errorf("Failed to register a consumer for queue %s: %s", queue, err)
		return
	}

	for msg := range msgs {
		var decodedMsg types.CommandMessage
		if err := json.NewDecoder(bytes.NewReader(msg.Body)).Decode(&decodedMsg); err != nil {
			logrus.Errorf("Failed to decode message: %s", err)
			msg.Nack(false, true) // requeue the message if decoding fails
			continue
		}

		s.mu.Lock()
		for _, agentID := range s.subscriptions[queue] {
			if client, exists := s.clients[agentID]; exists {
				if err := client.Send(&pb.Response{Result: decodedMsg.Command, CommandID: decodedMsg.CommandID, RoutingKey: decodedMsg.RoutingKey}); err == nil {
					logrus.Infof("Sent command: %s to agent %s", decodedMsg.Command, agentID)
					msg.Ack(false)
				} else {
					msg.Nack(false, true) // requeue the message if send fails
				}
			}
		}
		s.mu.Unlock()
	}
}
