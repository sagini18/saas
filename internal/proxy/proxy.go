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
		log.Printf("Received command: %s", cmd.Command.Command)
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
	conn, err := amqp.Dial("amqp://guest:guest@localhost:5672/")
	if err != nil {
		log.Fatalf("Failed to connect to RabbitMQ: %s", err)
	}
	defer conn.Close()

	ch, err := conn.Channel()
	if err != nil {
		log.Fatalf("Failed to open a channel: %s", err)
	}
	defer ch.Close()

	for {
		for queue := range s.subscriptions {
			go func(queue string) {
				_, err = ch.QueueDeclare(
					"k8s-1-command", // name
					true,            // durable
					false,           // delete when unused
					false,           // exclusive
					false,           // no-wait
					nil,             // arguments
				)
				if err != nil {
					log.Fatalf("%s: %s", "Failed to declare a queue", err)
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
					log.Printf("Failed to register a consumer for queue %s: %s", queue, err)
					return
				}
				for msg := range msgs {
					var decodedMsg types.CommandMessage
					if err := json.NewDecoder(bytes.NewReader(msg.Body)).Decode(&decodedMsg); err != nil {
						log.Printf("Failed to decode message: %s", err)
						msg.Nack(false, true) // requeue the message if decoding fails
						continue
					}

					for _, agentID := range s.subscriptions[queue] {
						if client, exists := s.clients[agentID]; exists {
							if err := client.Send(&pb.Response{Result: decodedMsg.Command, CommandID: decodedMsg.CommandID, RoutingKey: decodedMsg.RoutingKey}); err == nil {
								msg.Ack(false)
							} else {
								log.Printf("Failed to send message to client %s: %s", agentID, err)
								msg.Nack(false, true) // requeue the message if send fails
							}
						}
					}
				}
			}(queue)
		}
		time.Sleep(1 * time.Second)
	}
}
