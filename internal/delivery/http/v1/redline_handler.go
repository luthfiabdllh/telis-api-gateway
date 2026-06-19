package v1

import (
	"io/ioutil"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"telis-api-gateway/internal/domain"
)

type RedlineHandler struct {
	redlineUsecase domain.RedlineUsecase
}

func NewRedlineHandler(r *gin.RouterGroup, uc domain.RedlineUsecase) {
	handler := &RedlineHandler{
		redlineUsecase: uc,
	}
	// Note: Authentication and Role authorization middleware should be attached to the router group 'r'
	r.POST("/redlines", handler.UploadRedline)
	r.GET("/redlines/:id", handler.GetRedline)
}

func (h *RedlineHandler) UploadRedline(c *gin.Context) {
	userIDStr, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	userID, err := uuid.Parse(userIDStr.(string))
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid user ID token"})
		return
	}

	sourceFileHeader, err := c.FormFile("source_file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "source_file is required"})
		return
	}

	targetFileHeader, err := c.FormFile("target_file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "target_file is required"})
		return
	}

	sourceFile, _ := sourceFileHeader.Open()
	defer sourceFile.Close()
	sourceBytes, _ := ioutil.ReadAll(sourceFile)

	targetFile, _ := targetFileHeader.Open()
	defer targetFile.Close()
	targetBytes, _ := ioutil.ReadAll(targetFile)

	job, err := h.redlineUsecase.CreateRedlineJob(c.Request.Context(), userID, sourceBytes, targetBytes, sourceFileHeader.Filename, targetFileHeader.Filename)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusAccepted, gin.H{
		"message": "Redlining job started successfully",
		"job_id":  job.ID,
		"status":  job.Status,
	})
}

func (h *RedlineHandler) GetRedline(c *gin.Context) {
	jobIDStr := c.Param("id")
	jobID, err := uuid.Parse(jobIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid Job ID format"})
		return
	}

	job, err := h.redlineUsecase.GetRedlineJob(c.Request.Context(), jobID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Redline job not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"job_id":          job.ID,
		"status":          job.Status,
		"analysis_result": job.AnalysisResult,
		"created_at":      job.CreatedAt,
		"updated_at":      job.UpdatedAt,
	})
}
