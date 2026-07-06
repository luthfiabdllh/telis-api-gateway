package v1

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"bytes"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/ledongthuc/pdf"

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
		chatRoutes.POST("/extract-text", handler.ExtractText)
	}
}

type ChatPayload struct {
	SessionID       string   `json:"session_id" binding:"required"`
	Message         string   `json:"message" binding:"required"`
	DocumentFilters []string `json:"document_filters"`
	LLMTemperature  float32  `json:"llm_temperature"`
	ContextData     string   `json:"context_data"`
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
	finalMessage := req.Message
	if req.ContextData != "" {
		finalMessage = fmt.Sprintf("%s\n\n--- DOKUMEN LAMPIRAN ---\n%s", req.Message, req.ContextData)
	}

	grpcReq := &pb.ChatRequest{
		SessionId:       req.SessionID,
		UserId:          userIDStr.(string),
		Message:         finalMessage,
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
	c.Writer.Flush()

	// Buffer to store AI full response
	fullAIResponse := ""
	var sourcesBytes []byte
	isSaved := false

	// Defer ensures the partial response is saved if client disconnects prematurely
	defer func() {
		if !isSaved && fullAIResponse != "" {
			fullAIResponse += "\n\n*[Teks terputus karena koneksi]*"
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
			// Save immediately before closing HTTP stream
			if !isSaved && fullAIResponse != "" {
				_ = h.chatUsecase.SaveMessage(context.Background(), req.SessionID, "ai", fullAIResponse, sourcesBytes)
				isSaved = true
			}
			writeSSE(w, "done", "[DONE]")
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
			if !isSaved && fullAIResponse != "" {
				_ = h.chatUsecase.SaveMessage(context.Background(), req.SessionID, "ai", fullAIResponse, sourcesBytes)
				isSaved = true
			}
			writeSSE(w, "done", "[DONE]")
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
// @Param page query int false "Nomor Halaman" default(1)
// @Param limit query int false "Jumlah data per halaman" default(20)
// @Param search query string false "Kata kunci pencarian judul obrolan"
// @Success 200 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{} "Unauthorized"
// @Router /chat/sessions [get]
func (h *ChatHandler) GetSessions(c *gin.Context) {
	userIDStr, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not found in token"})
		return
	}

	userID, _ := uuid.Parse(userIDStr.(string))

	// Pagination and Search params
	page := 1
	limit := 20
	search := c.Query("search")

	if pageStr := c.Query("page"); pageStr != "" {
		if parsedPage, err := strconv.Atoi(pageStr); err == nil && parsedPage > 0 {
			page = parsedPage
		}
	}

	if limitStr := c.Query("limit"); limitStr != "" {
		if parsedLimit, err := strconv.Atoi(limitStr); err == nil && parsedLimit > 0 {
			limit = parsedLimit
		}
	}

	sessions, total, err := h.chatUsecase.GetSessions(c.Request.Context(), userID, search, page, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Calculate hasNextPage
	hasNextPage := (int64(page) * int64(limit)) < total

	// Return array directly if there are no sessions to avoid nil JSON array (nil slice marshals to null in some cases, though gin handles it. Better to explicitly use empty array if nil, but keeping it simple as before: session handles it)
	if sessions == nil {
		sessions = make([]*domain.ChatSession, 0)
	}

	c.JSON(http.StatusOK, gin.H{
		"data": sessions,
		"meta": gin.H{
			"currentPage": page,
			"hasNextPage": hasNextPage,
			"totalData":   total,
		},
	})
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
// @Success 204 "No Content"
// @Failure 401 {object} map[string]interface{} "Unauthorized"
// @Failure 403 {object} map[string]interface{} "Forbidden (Bukan pemilik sesi)"
// @Failure 404 {object} map[string]interface{} "Not Found"
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
		if strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, gin.H{"error": "chat session not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusNoContent)
}

// ExtractText godoc
// @Summary Ekstrak Teks dari PDF
// @Description Mengekstrak teks dari file PDF yang diunggah secara ad-hoc untuk disisipkan ke dalam context LLM.
// @Tags Chat
// @Accept multipart/form-data
// @Produce json
// @Security BearerAuth
// @Param file formData file true "File PDF"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{} "Bad Request"
// @Failure 401 {object} map[string]interface{} "Unauthorized"
// @Router /chat/extract-text [post]
func (h *ChatHandler) ExtractText(c *gin.Context) {
	_, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not found in token"})
		return
	}

	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "file is required"})
		return
	}

	if file.Header.Get("Content-Type") != "application/pdf" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "only PDF format is supported"})
		return
	}

	f, err := file.Open()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to open file"})
		return
	}
	defer f.Close()

	// Parse PDF
	// Convert multipart.File to bytes.Reader for pdf library
	buf := bytes.NewBuffer(nil)
	if _, err := io.Copy(buf, f); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to read file content"})
		return
	}
	
	reader := bytes.NewReader(buf.Bytes())
	pdfReader, err := pdf.NewReader(reader, reader.Size())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to parse pdf file"})
		return
	}
	
	var textBuilder strings.Builder
	numPages := pdfReader.NumPage()
	for i := 1; i <= numPages; i++ {
		p := pdfReader.Page(i)
		if p.V.IsNull() {
			continue
		}
		
		content, _ := p.GetPlainText(nil)
		textBuilder.WriteString(content)
		textBuilder.WriteString("\n")
	}

	extractedText := strings.TrimSpace(textBuilder.String())
	if len(extractedText) > 10000 {
		// Truncate if it's too large to prevent blowing up the JSON payload and LLM context
		extractedText = extractedText[:10000] + "\n...[Teks terpotong karena terlalu panjang]..."
	}

	c.JSON(http.StatusOK, gin.H{
		"text": extractedText,
	})
}

