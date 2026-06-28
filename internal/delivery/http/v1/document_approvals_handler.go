package v1

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type RequestApprovalReq struct {
	ApproverID string `json:"approver_id" binding:"required"`
	Notes      string `json:"notes"`
}

type ReviewApprovalReq struct {
	Status string `json:"status" binding:"required"` // APPROVED, REJECTED
	Notes  string `json:"notes"`
}

// RequestApproval godoc
// @Summary Ajukan Persetujuan Dokumen
// @Tags Document
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Document ID"
// @Param request body RequestApprovalReq true "Data Approval"
// @Success 201 {object} domain.ApprovalWorkflow
// @Router /documents/{id}/approvals [post]
func (h *DocumentHandler) RequestApproval(c *gin.Context) {
	docID := c.Param("id")
	userID := c.GetString("user_id")

	var req RequestApprovalReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	approval, err := h.docUsecase.RequestApproval(c.Request.Context(), docID, userID, req.ApproverID, req.Notes)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, approval)
}

// ReviewApproval godoc
// @Summary Review/Proses Persetujuan Dokumen
// @Tags Document
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Document ID"
// @Param aid path string true "Approval ID"
// @Param request body ReviewApprovalReq true "Keputusan Approval"
// @Success 200 {object} map[string]interface{}
// @Router /documents/{id}/approvals/{aid} [put]
func (h *DocumentHandler) ReviewApproval(c *gin.Context) {
	approvalID := c.Param("aid")
	userID := c.GetString("user_id")

	var req ReviewApprovalReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.docUsecase.ReviewApproval(c.Request.Context(), approvalID, userID, req.Status, req.Notes); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "approval updated"})
}

// GetDocumentApprovals godoc
// @Summary Dapatkan Riwayat Persetujuan Dokumen
// @Tags Document
// @Produce json
// @Security BearerAuth
// @Param id path string true "Document ID"
// @Success 200 {array} domain.ApprovalWorkflow
// @Router /documents/{id}/approvals [get]
func (h *DocumentHandler) GetDocumentApprovals(c *gin.Context) {
	docID := c.Param("id")

	approvals, err := h.docUsecase.GetDocumentApprovals(c.Request.Context(), docID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, approvals)
}
