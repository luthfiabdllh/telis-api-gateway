package v1

import (
	"fmt"
	"net/http"
	"time"

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
		docRoutes.GET("", handler.List)
		docRoutes.GET("/", handler.List)
		docRoutes.GET("/metadata-options", handler.GetMetadataOptions)
		docRoutes.GET("/:id", handler.GetByID)
		docRoutes.GET("/:id/download", handler.Download)
		docRoutes.GET("/:id/summarize", handler.Summarize) // Phase 1
		docRoutes.GET("/:id/clauses", handler.GetClauses) // Phase 2

		docRoutes.POST("/upload", handler.Upload)
		docRoutes.PATCH("/:id/metadata", handler.UpdateMetadata) // Phase 1
		docRoutes.DELETE("/:id", handler.Delete)
		docRoutes.POST("/:id/deprecate", handler.Deprecate)
		docRoutes.POST("/:id/restore", handler.Restore)
		docRoutes.PUT("/:id/rename", handler.Rename)
		docRoutes.PUT("/:id/move", handler.Move)
		
		// Phase 3 Approvals
		docRoutes.POST("/:id/approvals", handler.RequestApproval)
		docRoutes.PUT("/:id/approvals/:aid", handler.ReviewApproval)
		docRoutes.GET("/:id/approvals", handler.GetDocumentApprovals)
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

	// Poll database for up to 60 seconds to wait for ingestion result
	// This makes the upload "fail" if any ingestion step fails, as requested.
	timeout := 60
	for i := 0; i < timeout; i++ {
		doc, err := h.docUsecase.GetDocumentByID(c.Request.Context(), documentID)
		if err == nil {
			if doc.Status == "COMPLETED" {
				c.JSON(http.StatusOK, gin.H{
					"message":     "document uploaded and processed successfully",
					"document_id": documentID,
				})
				return
			}
			if doc.Status == "FAILED" {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error":       "document processing failed during ingestion pipeline",
					"document_id": documentID,
				})
				return
			}
		}
		
		time.Sleep(1 * time.Second)
	}

	c.JSON(http.StatusAccepted, gin.H{
		"message":     "document uploaded but still processing (timeout waiting for completion)",
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

// Restore godoc
// @Summary Pulihkan Dokumen
// @Description Mengembalikan dokumen yang usang (deprecated) agar diproses kembali oleh AI.
// @Tags Document
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "ID Dokumen"
// @Success 202 {object} map[string]interface{} "Perintah restore masuk antrean"
// @Failure 400 {object} map[string]interface{} "Bad Request"
// @Failure 401 {object} map[string]interface{} "Unauthorized"
// @Failure 500 {object} map[string]interface{} "Internal Server Error"
// @Router /documents/{id}/restore [post]
func (h *DocumentHandler) Restore(c *gin.Context) {
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

	err := h.docUsecase.RestoreDocument(c.Request.Context(), documentID, userID.(string))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusAccepted, gin.H{
		"message":     "document restore queued",
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
		Limit:        limit,
		Offset:       offset,
		Search:       c.Query("search"),
		Status:       c.Query("status"),
		DocumentType: c.Query("document_type"),
		RiskLevel:    c.Query("risk_level"),
		VendorName:   c.Query("vendor_name"),
		BusinessUnit: c.Query("business_unit"),
		SortBy:       c.Query("sort_by"),
		SortOrder:    c.Query("sort_order"),
	}

	if isDepStr := c.Query("is_deprecated"); isDepStr != "" {
		isDep := isDepStr == "true"
		filter.IsDeprecated = &isDep
	}

	isGlobal := c.Query("is_global") == "true"
	if !isGlobal {
		if fID := c.Query("folder_id"); fID != "" {
			filter.FolderID = &fID
		}
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
// @Param view query bool false "Jika true, tampilkan inline (tidak didownload)"
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

	if c.Query("view") == "true" {
		c.Header("Content-Type", "application/pdf")
		c.File(fullPath)
	} else {
		c.FileAttachment(fullPath, filename)
	}
}

// Rename godoc
// @Summary Ganti Nama Dokumen
// @Description Mengganti nama sebuah dokumen berdasarkan ID (otomatis ditambah .pdf jika belum ada).
// @Tags Document
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "ID Dokumen"
// @Param request body map[string]interface{} true "Payload Ganti Nama (name)"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{} "Bad Request"
// @Failure 404 {object} map[string]interface{} "Not Found"
// @Failure 500 {object} map[string]interface{} "Internal Server Error"
// @Router /documents/{id}/rename [put]
func (h *DocumentHandler) Rename(c *gin.Context) {
	documentID := c.Param("id")
	var req struct {
		Name string `json:"name" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err := h.docUsecase.RenameDocument(c.Request.Context(), documentID, req.Name)
	if err != nil {
		if err.Error() == "document not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "document renamed successfully",
	})
}

// Move godoc
// @Summary Pindahkan Dokumen
// @Description Memindahkan dokumen ke folder lain berdasarkan ID.
// @Tags Document
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "ID Dokumen"
// @Param request body map[string]interface{} true "Payload Pindah (folder_id)"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{} "Bad Request"
// @Failure 500 {object} map[string]interface{} "Internal Server Error"
// @Router /documents/{id}/move [put]
func (h *DocumentHandler) Move(c *gin.Context) {
	documentID := c.Param("id")
	var req struct {
		FolderID *string `json:"folder_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err := h.docUsecase.MoveDocument(c.Request.Context(), documentID, req.FolderID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "document moved successfully",
	})
}

// Summarize godoc
// @Summary Buat Ringkasan Dokumen
// @Description Menghasilkan ringkasan terstruktur dari dokumen legal.
//
//				Menggunakan hybrid cache-first: jika ringkasan sudah ada di DB, langsung return;
//				jika belum, generate via LLM (~5-10 detik) lalu simpan ke DB.
//
// @Tags Document
// @Produce json
// @Security BearerAuth
// @Param id path string true "ID Dokumen"
// @Param force query bool false "Force regenerate summary (bypass cache)"
// @Success 200 {object} map[string]interface{} "Ringkasan dokumen"
// @Failure 404 {object} map[string]interface{} "Dokumen tidak ditemukan"
// @Failure 500 {object} map[string]interface{} "Internal Server Error"
// @Router /documents/{id}/summarize [get]
func (h *DocumentHandler) Summarize(c *gin.Context) {
	documentID := c.Param("id")
	force := c.Query("force") == "true"

	result, err := h.docUsecase.SummarizeDocument(c.Request.Context(), documentID, force)
	if err != nil {
		if err.Error() == "document not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": "dokumen tidak ditemukan"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}

// UpdateRichMetadata godoc
// @Summary Update Metadata AI (Manual)
// @Description Memperbarui hasil ekstraksi AI secara manual.
// @Tags Document
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "ID Dokumen"
// @Param body body domain.DocumentRichMetadata true "Rich Metadata"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /documents/{id}/metadata [patch]
func (h *DocumentHandler) UpdateMetadata(c *gin.Context) {
	documentID := c.Param("id")
	var req domain.DocumentRichMetadata
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err := h.docUsecase.UpdateRichMetadata(c.Request.Context(), documentID, req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "metadata updated successfully"})
}

// GetMetadataOptions godoc
// @Summary Ambil opsi metadata
// @Description Mengambil daftar vendor dan business unit unik untuk dropdown filter
// @Tags Document
// @Produce json
// @Security BearerAuth
// @Success 200 {object} domain.MetadataOptions
// @Failure 500 {object} map[string]interface{}
// @Router /documents/metadata-options [get]
func (h *DocumentHandler) GetMetadataOptions(c *gin.Context) {
	opts, err := h.docUsecase.GetMetadataOptions(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, opts)
}


// GetClauses godoc
// @Summary Ambil Klausul Dokumen
// @Description Mengambil daftar klausul dan skor risiko dari dokumen (Phase 2).
// @Tags Document
// @Produce json
// @Security BearerAuth
// @Param id path string true "ID Dokumen"
// @Success 200 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{} "Unauthorized"
// @Failure 500 {object} map[string]interface{} "Internal Server Error"
// @Router /documents/{id}/clauses [get]
func (h *DocumentHandler) GetClauses(c *gin.Context) {
	documentID := c.Param("id")
	if documentID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "document id is required"})
		return
	}

	clauses, err := h.docUsecase.GetDocumentClauses(c.Request.Context(), documentID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": clauses,
	})
}
