package main

import (
	"context"
	"log"

	pb "github.com/sagini18/saas/proto"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)
type CommandResponse struct {
    Response   string `json:"response"`
	CommandID  string `json:"commandID"`
    RoutingKey string `json:"routingKey"`
}

func main() {
	// Subscribe to the queue
	agentID := "unique-agent-id"
	queue := "k8s-1-command" 

	for {
        err := runAgent(agentID, queue)
        if err != nil {
            log.Printf("Agent error: %v. Reconnecting in 5 seconds...", err)
            time.Sleep(5 * time.Second)
        }
    } 
}

func runAgent(agentID string, queue string) error{
	conn, err := grpc.NewClient("proxy-address:50051", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		logrus.Error("did not connect: %v", err)
		return err
	}
	defer conn.Close()
	client := pb.NewProxyClient(conn)

	subReq := &pb.SubscriptionRequest{
		AgentId: agentID,
		Queue:   queue,
	}
	ctx := metadata.AppendToOutgoingContext(context.Background(), "authorization", "valid-token")
    _, err = client.Subscribe(ctx, subReq)
    if err != nil {
        logrus.Error("could not subscribe: %v", err)
		retrun err
    }

	// Stream commands
	streamCtx := metadata.AppendToOutgoingContext(context.Background(), "authorization", "valid-token")
    stream, err := client.StreamCommands(streamCtx)
    if err != nil {
        logrus.Error("could not stream: %v", err)
		retrun err
    }

	initialReq := pb.InitialRequest{
		AgentId: agentID,
		Command: &pb.Command{Command: ""},
	}
	if err := stream.Send(&initialReq); err != nil {
		logrus.Error("could not send initial request: %v", err)
		return err
	}

	for {
		in, err := stream.Recv()
		if err != nil {
			logrus.Error("Failed to receive a command: %v", err)
			return err
		}

		config, err := rest.InClusterConfig()
		if err != nil {
			logrus.Error("Failed to create in-cluster config: %v", err)
			return err
		}

		clientset, err := kubernetes.NewForConfig(config)
		logrus.Info(clientset)
		if err != nil {
			logrus.Error("Failed to create clientset: %v", err)
			return err
		}

		// Execute the command using clientset
		log.Printf("Executing command: %s", in.Result)

		response := CommandResponse{
            Response:   "command executed successfully",
			CommandId:  in.CommandId,
            RoutingKey: queue,
        }

        // Send the response back to the server
        respBytes, err := json.Marshal(response)
        if err != nil {
            logrus.Error("Failed to marshal response: %v", err)
			return err
        }

        _, err = http.Post("http://saas-server:5050/api/v1/responses", "application/json", bytes.NewBuffer(respBytes))
        if err != nil {
            logrus.Error("Failed to send response: %v", err)
			return err
        }
	}
}
