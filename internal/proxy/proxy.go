package main

import (
	"context"
	"log"
	"net"
	"sync"

	pb "github.com/sagini18/saas/proto"
	"github.com/streadway/amqp"
	"google.golang.org/grpc"
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
        return nil, grpc.Errorf(grpc.Code(403), "Forbidden")
    }

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
		log.Printf("Received command: %s", cmd.Command.Command)
	}
}

func main() {
	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

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

	for queue := range s.subscriptions {
		go func(queue string) {
			msgs, err := ch.Consume(
				queue, // queue
				"",    // consumer
				false, // auto-ack (disable auto-ack for manual ack)
				false, // exclusive
				false, // no-local
				false, // no-wait
				nil,   // args
			)
			if err != nil {
				log.Fatalf("Failed to register a consumer: %s", err)
			}

			for msg := range msgs {
				s.mu.Lock()
				for _, agentID := range s.subscriptions[queue] {
					if client, exists := s.clients[agentID]; exists {
						if err := client.Send(&pb.Response{Result: string(msg.Body)}); err == nil {
							msg.Ack(false)
						} else {
							msg.Nack(false, true) // requeue the message if send fails
						}
					}
				}
				s.mu.Unlock()
			}
		}(queue)
	}
}
