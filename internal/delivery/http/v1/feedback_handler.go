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
	Rating    int    `json:"rating" binding:"required"`
	Comment   string `json:"comment"`
}

func (h *FeedbackHandler) SubmitFeedback(c *gin.Context) {
	// Extract userID from context (set by AuthMiddleware)
	userIDStr, exists := c.Get("userID")
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
