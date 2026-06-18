package v1

import (
	"fmt"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"

	grpcClient "telis-api-gateway/internal/infrastructure/grpc"
	"telis-api-gateway/pb"
)

type ChatHandler struct {
	agentClient grpcClient.AgentClient
}

func NewChatHandler(r *gin.RouterGroup, agentClient grpcClient.AgentClient) {
	handler := &ChatHandler{
		agentClient: agentClient,
	}

	chatRoutes := r.Group("/chat")
	{
		chatRoutes.POST("/stream", handler.ChatStream)
	}
}

type ChatPayload struct {
	SessionID       string   `json:"session_id" binding:"required"`
	Message         string   `json:"message" binding:"required"`
	DocumentFilters []string `json:"document_filters"`
	LLMTemperature  float32  `json:"llm_temperature"`
}

func (h *ChatHandler) ChatStream(c *gin.Context) {
	// 1. Validate JWT
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not found in token"})
		return
	}

	// 2. Parse payload
	var req ChatPayload
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 3. Forward to Agent via gRPC
	grpcReq := &pb.ChatRequest{
		SessionId:       req.SessionID,
		UserId:          userID.(string),
		Message:         req.Message,
		DocumentFilters: req.DocumentFilters,
		LlmTemperature:  req.LLMTemperature,
	}

	stream, err := h.agentClient.ChatStream(c.Request.Context(), grpcReq)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": fmt.Sprintf("failed to connect to agent service: %v", err)})
		return
	}

	// 4. Set Headers for Server-Sent Events (SSE)
	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")
	c.Writer.Header().Set("Transfer-Encoding", "chunked")

	// 5. Read from gRPC Stream and write to HTTP SSE
	c.Stream(func(w io.Writer) bool {
		resp, err := stream.Recv()
		if err == io.EOF {
			return false // End of stream
		}
		if err != nil {
			c.SSEvent("error", err.Error())
			return false
		}

		c.SSEvent("message", resp.ContentChunk)

		// Check if it's final
		if resp.IsFinal {
			c.SSEvent("done", "[DONE]")
			return false
		}

		return true // continue streaming
	})
}
