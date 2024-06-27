package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/sagini18/saas/internal/types"
	pb "github.com/sagini18/saas/proto"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
)

func main() {
	agentID := "unique-agent-id"
	queue := "k8s-1-command"

	for {
		err := runAgent(agentID, queue)
		if err != nil {
			logrus.Errorf("Agent error: %v. Reconnecting in 5 seconds...", err)
			time.Sleep(5 * time.Second)
			continue
		}
	}
}

func runAgent(agentID string, queue string) error {
	conn, err := grpc.NewClient("localhost:50051", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		logrus.Errorf("did not connect: %v", err)
		return err
	}
	defer conn.Close()
	client := pb.NewProxyClient(conn)

	subReq := &pb.SubscriptionRequest{
		AgentId: agentID,
		Queue:   queue,
	}
	ctx := metadata.AppendToOutgoingContext(context.Background(), "authorization", "valid-token")
	res, err := client.Subscribe(ctx, subReq)
	if err != nil {
		logrus.Errorf("could not subscribe: %v", err)
		return err
	}
	logrus.Info(res.Message)

	// Stream commands
	for {
		streamCtx := metadata.AppendToOutgoingContext(context.Background(), "authorization", "valid-token")
		stream, err := client.StreamCommands(streamCtx)
		if err != nil {
			logrus.Errorf("could not stream: %v", err)
			time.Sleep(5 * time.Second)
			continue
		}

		initialReq := pb.InitialRequest{
			AgentId: agentID,
			Command: &pb.Command{Command: ""}, // need to delete this line
		}
		if err := stream.Send(&initialReq); err != nil {
			logrus.Errorf("could not send initial request: %v", err)
			return err
		}

		for {
			in, err := stream.Recv()
			if err != nil {
				logrus.Errorf("Failed to receive a command: %v", err)
				break // Break inner loop to reconnect the stream
			}

			response := types.CommandResponse{
				Response:   "command executed successfully",
				CommandID:  in.CommandID,
				RoutingKey: in.RoutingKey,
			}
			logrus.Infof("Received command: %s with commandID: %s", in.Result, in.CommandID)

			// Send the response back to the server
			respBytes, err := json.Marshal(response)
			if err != nil {
				logrus.Errorf("Failed to marshal response: %v", err)
				return err
			}
			fmt.Println(" Sending response: ", string(respBytes))

			_, err = http.Post("http://localhost:5050/api/v1/responses", "application/json", bytes.NewBuffer(respBytes))
			if err != nil {
				logrus.Errorf("Failed to send response: %v", err)
				return err
			}
		}
		logrus.Info("Reconnecting stream in 5 seconds...")
		time.Sleep(5 * time.Second)
	}
}
