package grpc

import (
	"context"
	"log"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"telis-api-gateway/pb"
)

type AgentClient interface {
	ChatStream(ctx context.Context, req *pb.ChatRequest) (pb.AgentService_ChatStreamClient, error)
	Close() error
}

type agentClient struct {
	conn   *grpc.ClientConn
	client pb.AgentServiceClient
}

func NewAgentClient(url string) (AgentClient, error) {
	conn, err := grpc.NewClient(url, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}

	client := pb.NewAgentServiceClient(conn)
	log.Printf("Connected to Agent Service via gRPC at %s", url)

	return &agentClient{
		conn:   conn,
		client: client,
	}, nil
}

func (c *agentClient) ChatStream(ctx context.Context, req *pb.ChatRequest) (pb.AgentService_ChatStreamClient, error) {
	return c.client.ChatStream(ctx, req)
}

func (c *agentClient) Close() error {
	return c.conn.Close()
}
