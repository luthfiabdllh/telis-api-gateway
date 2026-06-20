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

// Upload godoc
// @Summary Unggah Dokumen PDF
// @Description Mengunggah dokumen PDF untuk diproses oleh Celery Ingestion Worker.
// @Tags Document
// @Accept multipart/form-data
// @Produce json
// @Security BearerAuth
// @Param file formData file true "File PDF Dokumen Hukum"
// @Param replaces_document_id formData string false "ID Dokumen versi lama jika dokumen ini adalah revisi"
// @Success 202 {object} map[string]interface{} "Diterima dan masuk antrean"
// @Failure 400 {object} map[string]interface{} "Bad Request"
// @Failure 401 {object} map[string]interface{} "Unauthorized"
// @Failure 500 {object} map[string]interface{} "Internal Server Error"
// @Router /documents/upload [post]
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

// Delete godoc
// @Summary Hapus Dokumen Permanen
// @Description Menghapus dokumen PDF, Vektor di Qdrant, dan entitas di Neo4j secara permanen.
// @Tags Document
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "ID Dokumen"
// @Success 202 {object} map[string]interface{} "Perintah penghapusan masuk antrean"
// @Failure 400 {object} map[string]interface{} "Bad Request"
// @Failure 401 {object} map[string]interface{} "Unauthorized"
// @Failure 500 {object} map[string]interface{} "Internal Server Error"
// @Router /documents/{id} [delete]
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

// Deprecate godoc
// @Summary Usangkan Dokumen
// @Description Mengusangkan dokumen lama (mencabut vektor & graph) tanpa menghapus file fisiknya.
// @Tags Document
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "ID Dokumen"
// @Success 202 {object} map[string]interface{} "Perintah pengusangan masuk antrean"
// @Failure 400 {object} map[string]interface{} "Bad Request"
// @Failure 401 {object} map[string]interface{} "Unauthorized"
// @Failure 500 {object} map[string]interface{} "Internal Server Error"
// @Router /documents/{id}/deprecate [post]
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
