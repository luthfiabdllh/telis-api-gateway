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
		docRoutes.DELETE("/:id", handler.Delete)
		docRoutes.POST("/:id/deprecate", handler.Deprecate)
	}
}

func (h *DocumentHandler) Upload(c *gin.Context) {
	// Parse multipart form
	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "file is required"})
		return
	}

	replacesDocumentID := c.PostForm("replaces_document_id")

	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not found in token"})
		return
	}

	// Call usecase
	documentID, err := h.docUsecase.UploadDocument(c.Request.Context(), userID.(string), file, replacesDocumentID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusAccepted, gin.H{
		"message":     "document uploaded successfully and queued for processing",
		"document_id": documentID,
	})
}

func (h *DocumentHandler) Delete(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not found in token"})
		return
	}

	documentID := c.Param("id")
	if documentID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "document id is required"})
		return
	}

	err := h.docUsecase.DeleteDocument(c.Request.Context(), documentID, userID.(string))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusAccepted, gin.H{
		"message":     "document deletion queued",
		"document_id": documentID,
	})
}

func (h *DocumentHandler) Deprecate(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not found in token"})
		return
	}

	documentID := c.Param("id")
	if documentID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "document id is required"})
		return
	}

	err := h.docUsecase.DeprecateDocument(c.Request.Context(), documentID, userID.(string))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusAccepted, gin.H{
		"message":     "document deprecation queued",
		"document_id": documentID,
	})
}
