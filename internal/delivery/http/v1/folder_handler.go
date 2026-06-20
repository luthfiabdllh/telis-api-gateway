package v1

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"telis-api-gateway/internal/domain"
)

type FolderHandler struct {
	folderUsecase domain.FolderUsecase
}

func NewFolderHandler(r *gin.RouterGroup, folderUsecase domain.FolderUsecase) {
	handler := &FolderHandler{
		folderUsecase: folderUsecase,
	}

	folderRoutes := r.Group("/folders")
	{
		folderRoutes.POST("", handler.Create)
		folderRoutes.GET("", handler.List)
		folderRoutes.PUT("/:id", handler.Rename)
		folderRoutes.DELETE("/:id", handler.Delete)
	}
}

// Create godoc
// @Summary Buat Folder Baru
// @Description Membuat folder baru. Bisa disarangkan ke dalam folder lain menggunakan parent_id.
// @Tags Folder
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body map[string]interface{} true "Payload Buat Folder (name, parent_id)"
// @Success 201 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{} "Bad Request"
// @Failure 500 {object} map[string]interface{} "Internal Server Error"
// @Router /folders [post]
func (h *FolderHandler) Create(c *gin.Context) {
	var req struct {
		Name     string  `json:"name" binding:"required"`
		ParentID *string `json:"parent_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID := c.GetString("user_id") // From JWT middleware

	folder, err := h.folderUsecase.CreateFolder(c.Request.Context(), userID, req.Name, req.ParentID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "folder created successfully",
		"data":    folder,
	})
}

// List godoc
// @Summary Daftar Folder
// @Description Mengambil daftar folder. Jika parent_id dikosongkan, akan mengambil root folder.
// @Tags Folder
// @Produce json
// @Security BearerAuth
// @Param parent_id query string false "ID Folder Induk (kosongkan untuk root)"
// @Success 200 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{} "Internal Server Error"
// @Router /folders [get]
func (h *FolderHandler) List(c *gin.Context) {
	var parentID *string
	if pid := c.Query("parent_id"); pid != "" {
		parentID = &pid
	}

	folders, err := h.folderUsecase.GetFolders(c.Request.Context(), parentID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": folders,
	})
}

// Rename godoc
// @Summary Ganti Nama Folder
// @Description Mengganti nama sebuah folder berdasarkan ID.
// @Tags Folder
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "ID Folder"
// @Param request body map[string]interface{} true "Payload Ganti Nama (name)"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{} "Bad Request"
// @Failure 404 {object} map[string]interface{} "Not Found"
// @Failure 500 {object} map[string]interface{} "Internal Server Error"
// @Router /folders/{id} [put]
func (h *FolderHandler) Rename(c *gin.Context) {
	folderID := c.Param("id")
	var req struct {
		Name string `json:"name" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err := h.folderUsecase.RenameFolder(c.Request.Context(), folderID, req.Name)
	if err != nil {
		if err.Error() == "folder not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "folder renamed successfully",
	})
}

// Delete godoc
// @Summary Hapus Folder (Cascading Delete)
// @Description Menghapus folder, seluruh sub-folder di dalamnya, beserta memicu penghapusan dokumen secara permanen dari Storage, Qdrant, dan Neo4j.
// @Tags Folder
// @Produce json
// @Security BearerAuth
// @Param id path string true "ID Folder"
// @Success 200 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{} "Internal Server Error"
// @Router /folders/{id} [delete]
func (h *FolderHandler) Delete(c *gin.Context) {
	folderID := c.Param("id")
	userID := c.GetString("user_id")

	err := h.folderUsecase.DeleteFolder(c.Request.Context(), folderID, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "folder and its contents deleted successfully",
	})
}