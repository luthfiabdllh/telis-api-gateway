package v1

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"telis-api-gateway/internal/domain"
)

type FeedbackHandler struct {
	usecase domain.FeedbackUsecase
}

func NewFeedbackHandler(rg *gin.RouterGroup, usecase domain.FeedbackUsecase) {
	handler := &FeedbackHandler{
		usecase: usecase,
	}

	feedbackRoute := rg.Group("/chat")
	{
		feedbackRoute.POST("/feedback", handler.SubmitFeedback)
	}
}

type submitFeedbackRequest struct {
	MessageID string `json:"message_id" binding:"required"`
	Rating    int    `json:"rating"`
	Comment   string `json:"comment"`
}

// SubmitFeedback godoc
// @Summary Kirim Umpan Balik (HITL)
// @Description Menyimpan rating jempol dan komentar dari user untuk evaluasi RAG (Data Flywheel). UserID diambil secara otomatis dari token JWT.
// @Tags Chat
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body v1.submitFeedbackRequest true "Payload Feedback"
// @Success 201 {object} map[string]interface{} "Feedback tersimpan"
// @Failure 400 {object} map[string]interface{} "Bad Request"
// @Failure 401 {object} map[string]interface{} "Unauthorized"
// @Router /chat/feedback [post]
func (h *FeedbackHandler) SubmitFeedback(c *gin.Context) {
	// Extract userID from context (set by AuthMiddleware)
	userIDStr, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User ID not found in context"})
		return
	}

	userID, err := uuid.Parse(userIDStr.(string))
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid User ID"})
		return
	}

	var req submitFeedbackRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	feedback, err := h.usecase.SubmitFeedback(c.Request.Context(), req.MessageID, userID, req.Rating, req.Comment)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "Feedback submitted successfully",
		"data":    feedback,
	})
}
