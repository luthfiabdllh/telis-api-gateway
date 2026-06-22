package v1

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"telis-api-gateway/internal/domain"
	grpcClient "telis-api-gateway/internal/infrastructure/grpc"
	"telis-api-gateway/pb"
)

type ChatHandler struct {
	agentClient grpcClient.AgentClient
	chatUsecase domain.ChatUsecase
}

func NewChatHandler(r *gin.RouterGroup, agentClient grpcClient.AgentClient, chatUsecase domain.ChatUsecase) {
	handler := &ChatHandler{
		agentClient: agentClient,
		chatUsecase: chatUsecase,
	}

	chatRoutes := r.Group("/chat")
	{
		chatRoutes.POST("/stream", handler.ChatStream)
		chatRoutes.GET("/sessions", handler.GetSessions)
		chatRoutes.GET("/sessions/:id/messages", handler.GetSessionMessages)
		chatRoutes.PUT("/sessions/:id/title", handler.RenameSession)
		chatRoutes.DELETE("/sessions/:id", handler.DeleteSession)
	}
}

type ChatPayload struct {
	SessionID       string   `json:"session_id" binding:"required"`
	Message         string   `json:"message" binding:"required"`
	DocumentFilters []string `json:"document_filters"`
	LLMTemperature  float32  `json:"llm_temperature"`
}

// ChatStream godoc
// @Summary Chat dengan Agen RAG (SSE)
// @Description Mengirim pesan ke AI Agent dan menerima respons secara streaming (Server-Sent Events).
// @Tags Chat
// @Accept json
// @Produce text/event-stream
// @Security BearerAuth
// @Param request body v1.ChatPayload true "Payload Chat"
// @Success 200 {string} string "Server-Sent Events stream"
// @Failure 400 {object} map[string]interface{} "Bad Request"
// @Failure 401 {object} map[string]interface{} "Unauthorized"
// @Failure 502 {object} map[string]interface{} "Bad Gateway (Agent mati)"
// @Router /chat/stream [post]
func (h *ChatHandler) ChatStream(c *gin.Context) {
	// 1. Validate JWT
	userIDStr, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not found in token"})
		return
	}

	userID, err := uuid.Parse(userIDStr.(string))
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user id"})
		return
	}

	// 2. Parse payload
	var req ChatPayload
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// [NEW] Ensure Session exists and save User Message
	if err := h.chatUsecase.EnsureSessionExists(c.Request.Context(), req.SessionID, userID, req.Message); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to initialize chat session"})
		return
	}
	if err := h.chatUsecase.SaveMessage(c.Request.Context(), req.SessionID, "user", req.Message, nil); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save user message"})
		return
	}

	// 3. Forward to Agent via gRPC
	grpcReq := &pb.ChatRequest{
		SessionId:       req.SessionID,
		UserId:          userIDStr.(string),
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

	// Buffer to store AI full response
	fullAIResponse := ""
	var sourcesBytes []byte
	isStreamCompleted := false

	// [NEW] Defer ensures the partial response is ALWAYS saved, even if the user refreshes/disconnects
	defer func() {
		if fullAIResponse != "" {
			if !isStreamCompleted {
				fullAIResponse += "\n\n*[Teks terputus karena koneksi]*"
			}
			_ = h.chatUsecase.SaveMessage(context.Background(), req.SessionID, "ai", fullAIResponse, sourcesBytes)
		}
	}()

	// Helper function for W3C-compliant SSE
	writeSSE := func(w io.Writer, event string, data string) {
		fmt.Fprintf(w, "event: %s\n", event)
		if len(data) > 0 {
			lines := strings.Split(data, "\n")
			for _, line := range lines {
				fmt.Fprintf(w, "data: %s\n", line)
			}
		}
		fmt.Fprintf(w, "\n")
	}

	// 5. Read from gRPC Stream and write to HTTP SSE
	c.Stream(func(w io.Writer) bool {
		resp, err := stream.Recv()
		if err == io.EOF {
			isStreamCompleted = true
			return false // End of stream
		}
		if err != nil {
			writeSSE(w, "error", err.Error())
			return false
		}

		if resp.EventType == "status" {
			b, _ := json.Marshal(resp.ContentChunk)
			writeSSE(w, "status", string(b))
			c.Writer.Flush()
			return true
		}

		if resp.EventType == "sources" {
			sourcesBytes = []byte(resp.ContentChunk)
			writeSSE(w, "sources", resp.ContentChunk)
			c.Writer.Flush()
			return true
		}

		b, _ := json.Marshal(resp.ContentChunk)
		writeSSE(w, "message", string(b))
		fullAIResponse += resp.ContentChunk

		// Check if it's final
		if resp.IsFinal || resp.EventType == "done" {
			writeSSE(w, "done", "[DONE]")
			isStreamCompleted = true
			return false
		}

		return true // continue streaming
	})
}

// GetSessions godoc
// @Summary Daftar Riwayat Obrolan
// @Description Mengambil daftar semua sesi chat milik pengguna yang sedang login.
// @Tags Chat
// @Produce json
// @Security BearerAuth
// @Success 200 {array} domain.ChatSession
// @Failure 401 {object} map[string]interface{} "Unauthorized"
// @Router /chat/sessions [get]
func (h *ChatHandler) GetSessions(c *gin.Context) {
	userIDStr, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not found in token"})
		return
	}

	userID, _ := uuid.Parse(userIDStr.(string))
	sessions, err := h.chatUsecase.GetSessions(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, sessions)
}

// GetSessionMessages godoc
// @Summary Histori Pesan Obrolan
// @Description Mengambil riwayat pesan di dalam sesi tertentu.
// @Tags Chat
// @Produce json
// @Security BearerAuth
// @Param id path string true "Session ID"
// @Success 200 {array} domain.ChatMessage
// @Failure 401 {object} map[string]interface{} "Unauthorized"
// @Failure 403 {object} map[string]interface{} "Forbidden (Bukan pemilik sesi)"
// @Failure 404 {object} map[string]interface{} "Sesi tidak ditemukan"
// @Router /chat/sessions/{id}/messages [get]
func (h *ChatHandler) GetSessionMessages(c *gin.Context) {
	userIDStr, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not found in token"})
		return
	}
	userID, _ := uuid.Parse(userIDStr.(string))

	sessionIDStr := c.Param("id")
	sessionID, err := uuid.Parse(sessionIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid session id"})
		return
	}

	messages, err := h.chatUsecase.GetMessages(c.Request.Context(), sessionID, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, messages)
}

type RenameSessionRequest struct {
	Title string `json:"title" binding:"required"`
}

// RenameSession godoc
// @Summary Ubah Judul Obrolan
// @Description Mengubah judul sesi obrolan.
// @Tags Chat
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Session ID"
// @Param request body v1.RenameSessionRequest true "Judul Baru"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{} "Bad Request"
// @Failure 401 {object} map[string]interface{} "Unauthorized"
// @Failure 403 {object} map[string]interface{} "Forbidden"
// @Router /chat/sessions/{id}/title [put]
func (h *ChatHandler) RenameSession(c *gin.Context) {
	userIDStr, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not found in token"})
		return
	}
	userID, _ := uuid.Parse(userIDStr.(string))

	sessionIDStr := c.Param("id")
	sessionID, err := uuid.Parse(sessionIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid session id"})
		return
	}

	var req RenameSessionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.chatUsecase.RenameSession(c.Request.Context(), sessionID, userID, req.Title); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "session title updated successfully"})
}

// DeleteSession godoc
// @Summary Hapus Sesi Obrolan
// @Description Menghapus seluruh sesi beserta riwayat pesannya.
// @Tags Chat
// @Produce json
// @Security BearerAuth
// @Param id path string true "Session ID"
// @Success 200 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{} "Unauthorized"
// @Failure 403 {object} map[string]interface{} "Forbidden (Bukan pemilik sesi)"
// @Router /chat/sessions/{id} [delete]
func (h *ChatHandler) DeleteSession(c *gin.Context) {
	userIDStr, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not found in token"})
		return
	}
	userID, _ := uuid.Parse(userIDStr.(string))

	sessionIDStr := c.Param("id")
	sessionID, err := uuid.Parse(sessionIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid session id"})
		return
	}

	if err := h.chatUsecase.DeleteSession(c.Request.Context(), sessionID, userID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "session deleted successfully"})
}
