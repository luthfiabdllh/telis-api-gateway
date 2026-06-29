package grpc

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"github.com/sony/gobreaker"

	"telis-api-gateway/pb"
	"telis-api-gateway/pkg/circuitbreaker"
)

type AgentClient interface {
	ChatStream(ctx context.Context, req *pb.ChatRequest) (pb.AgentService_ChatStreamClient, error)
	// Phase 1: Summarize a document — returns structured summary as JSON string
	SummarizeDocument(ctx context.Context, documentID string, documentType string) (string, error)
	Close() error
}

type agentClient struct {
	conn         *grpc.ClientConn
	client       pb.AgentServiceClient
	agentHTTPURL string // e.g. http://localhost:8001 — used for non-streaming endpoints
	cb           *gobreaker.CircuitBreaker
}

func NewAgentClient(url string) (AgentClient, error) {
	conn, err := grpc.NewClient(url, 
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithStatsHandler(otelgrpc.NewClientHandler()),
	)
	if err != nil {
		return nil, err
	}

	client := pb.NewAgentServiceClient(conn)
	log.Printf("Connected to Agent Service via gRPC at %s", url)

	// Derive HTTP URL from gRPC address for non-streaming calls
	// gRPC url: "agent-service:8001" → HTTP: "http://agent-service:8002-or-env"
	agentHTTPURL := os.Getenv("AGENT_HTTP_URL")
	if agentHTTPURL == "" {
		// Fallback: replace port 8001 with 8003 (interim HTTP endpoint on agent service)
		agentHTTPURL = "http://" + strings.Replace(url, ":8001", ":8003", 1)
	}

	return &agentClient{
		conn:         conn,
		client:       client,
		agentHTTPURL: agentHTTPURL,
		cb:           circuitbreaker.NewCB("agent-service"),
	}, nil
}

func (c *agentClient) ChatStream(ctx context.Context, req *pb.ChatRequest) (pb.AgentService_ChatStreamClient, error) {
	result, err := c.cb.Execute(func() (interface{}, error) {
		return c.client.ChatStream(ctx, req)
	})
	if err != nil {
		return nil, err
	}
	return result.(pb.AgentService_ChatStreamClient), nil
}

// SummarizeDocument calls the Agent Service HTTP endpoint (interim, until proto is regenerated).
// POST /summarize {"document_id": "...", "document_type": "..."}
// Returns JSON string e.g. {"pihak_terlibat": [...], "ringkasan_singkat": "..."}
func (c *agentClient) SummarizeDocument(ctx context.Context, documentID string, documentType string) (string, error) {
	body, _ := json.Marshal(map[string]string{
		"document_id":   documentID,
		"document_type": documentType,
	})

	result, err := c.cb.Execute(func() (interface{}, error) {
		reqHTTP, errHTTP := http.NewRequestWithContext(ctx, "POST",
			fmt.Sprintf("%s/summarize", c.agentHTTPURL),
			bytes.NewBuffer(body),
		)
		if errHTTP != nil {
			return "", fmt.Errorf("failed to build summarize request: %v", errHTTP)
		}
		reqHTTP.Header.Set("Content-Type", "application/json")
		// (OTEL propagation for HTTP would be needed here, but since it's just a fallback we skip it for now to focus on gRPC)

		resp, errHTTP := http.DefaultClient.Do(reqHTTP)
		if errHTTP != nil {
			return "", fmt.Errorf("summarize request failed: %v", errHTTP)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return "", fmt.Errorf("summarize returned status %d", resp.StatusCode)
		}

		resultBytes, errHTTP := io.ReadAll(resp.Body)
		if errHTTP != nil {
			return "", fmt.Errorf("failed to read summarize response: %v", errHTTP)
		}
		return string(resultBytes), nil
	})

	if err != nil {
		return "", err
	}

	return result.(string), nil
}

func (c *agentClient) Close() error {
	return c.conn.Close()
}
