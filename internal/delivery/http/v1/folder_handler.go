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
