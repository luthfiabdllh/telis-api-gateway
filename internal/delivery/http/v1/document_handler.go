package v1

import (
	"fmt"
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
		docRoutes.GET("/", handler.List)
		docRoutes.GET("/:id", handler.GetByID)
		docRoutes.GET("/:id/download", handler.Download)
		
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
	folderID := c.PostForm("folder_id")
	userID := c.GetString("user_id") // From JWT middleware

	// Call usecase
	documentID, err := h.docUsecase.UploadDocument(c.Request.Context(), userID, file, folderID, replacesDocumentID)
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

// List godoc
// @Summary Ambil Daftar Dokumen
// @Description Mengambil daftar dokumen dengan fitur paginasi dan pencarian.
// @Tags Document
// @Produce json
// @Security BearerAuth
// @Param limit query int false "Limit (default 10)"
// @Param offset query int false "Offset (default 0)"
// @Param search query string false "Search by filename"
// @Param status query string false "Filter by status"
// @Param is_deprecated query bool false "Filter by is_deprecated"
// @Success 200 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{} "Unauthorized"
// @Failure 500 {object} map[string]interface{} "Internal Server Error"
// @Router /documents [get]
func (h *DocumentHandler) List(c *gin.Context) {
	limit := 10
	offset := 0

	if l := c.Query("limit"); l != "" {
		fmt.Sscanf(l, "%d", &limit)
	}
	if o := c.Query("offset"); o != "" {
		fmt.Sscanf(o, "%d", &offset)
	}

	filter := domain.DocumentFilter{
		Limit:  limit,
		Offset: offset,
		Search: c.Query("search"),
		Status: c.Query("status"),
	}

	if isDepStr := c.Query("is_deprecated"); isDepStr != "" {
		isDep := isDepStr == "true"
		filter.IsDeprecated = &isDep
	}

	if fID := c.Query("folder_id"); fID != "" {
		filter.FolderID = &fID
	}

	docs, total, err := h.docUsecase.GetAllDocuments(c.Request.Context(), filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":  docs,
		"total": total,
	})
}

// GetByID godoc
// @Summary Detail Dokumen
// @Description Mengambil detail metadata 1 dokumen berdasarkan ID.
// @Tags Document
// @Produce json
// @Security BearerAuth
// @Param id path string true "ID Dokumen"
// @Success 200 {object} domain.Document
// @Failure 401 {object} map[string]interface{} "Unauthorized"
// @Failure 404 {object} map[string]interface{} "Not Found"
// @Failure 500 {object} map[string]interface{} "Internal Server Error"
// @Router /documents/{id} [get]
func (h *DocumentHandler) GetByID(c *gin.Context) {
	documentID := c.Param("id")
	doc, err := h.docUsecase.GetDocumentByID(c.Request.Context(), documentID)
	if err != nil {
		if err.Error() == "document not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, doc)
}

// Download godoc
// @Summary Unduh PDF Dokumen
// @Description Mengunduh file fisik PDF dari dokumen yang tersimpan di sistem.
// @Tags Document
// @Produce application/pdf
// @Security BearerAuth
// @Param id path string true "ID Dokumen"
// @Success 200 {file} file
// @Failure 401 {object} map[string]interface{} "Unauthorized"
// @Failure 404 {object} map[string]interface{} "Not Found"
// @Failure 500 {object} map[string]interface{} "Internal Server Error"
// @Router /documents/{id}/download [get]
func (h *DocumentHandler) Download(c *gin.Context) {
	documentID := c.Param("id")
	fullPath, filename, err := h.docUsecase.GetDocumentFilePath(c.Request.Context(), documentID)
	if err != nil {
		if err.Error() == "document not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.FileAttachment(fullPath, filename)
}
