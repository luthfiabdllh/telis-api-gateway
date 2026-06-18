package v1

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"telis-api-gateway/internal/domain"
)

type DocumentHandler struct {
	docUsecase domain.DocumentUsecase
}

func NewDocumentHandler(r *gin.RouterGroup, docUsecase domain.DocumentUsecase) {
	handler := &DocumentHandler{
		docUsecase: docUsecase,
	}

	docRoutes := r.Group("/documents")
	{
		docRoutes.POST("/upload", handler.Upload)
	}
}

func (h *DocumentHandler) Upload(c *gin.Context) {
	// Get user_id from middleware context
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not found in token"})
		return
	}

	// Parse multipart form
	fileHeader, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to retrieve file from form-data key 'file'"})
		return
	}

	documentID, err := h.docUsecase.UploadDocument(c.Request.Context(), userID.(string), fileHeader)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusAccepted, gin.H{
		"message":     "document uploaded successfully and queued for processing",
		"document_id": documentID,
	})
}
