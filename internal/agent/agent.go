package main

import (
	"context"
	"time"

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
		stream, err := client.StreamCommands(ctx)
		if err != nil {
			logrus.Errorf("could not stream: %v", err)
			time.Sleep(5 * time.Second)
			continue
		}

		initialReq := pb.CommandStream{
			AgentId: agentID,
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

			logrus.Infof("Received command: %s with commandID: %s", in.Response.Response, in.Response.CommandID)

			go func(in *pb.Response) {
				// Simulate processing the command
				time.Sleep(2 * time.Second)

				responseMsg := &pb.CommandStream{
					AgentId:  agentID,
					Response: &pb.Response{Response: "command executed successfully", CommandID: in.CommandID, RoutingKey: in.RoutingKey},
				}

				if err := stream.Send(responseMsg); err != nil {
					logrus.Errorf("Failed to send response: %v", err)
				}
				logrus.Infof("Sent response for commandID: %s", in.CommandID)
			}(in.Response)

			// response := types.CommandResponse{
			// 	Response:   "command executed successfully",
			// 	CommandID:  in.CommandID,
			// 	RoutingKey: in.RoutingKey,
			// }
			// logrus.Infof("Received command: %s with commandID: %s", in.Result, in.CommandID)

			// // Send the response back to the server
			// respBytes, err := json.Marshal(response)
			// if err != nil {
			// 	logrus.Errorf("Failed to marshal response: %v", err)
			// 	return err
			// }
			// fmt.Println(" Sending response: ", string(respBytes))

			// _, err = http.Post("http://localhost:5050/api/v1/responses", "application/json", bytes.NewBuffer(respBytes))
			// if err != nil {
			// 	logrus.Errorf("Failed to send response: %v", err)
			// 	return err
			// }
		}
		logrus.Info("Reconnecting stream in 5 seconds...")
		time.Sleep(5 * time.Second)
	}
}
