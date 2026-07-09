package v1

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"telis-api-gateway/internal/domain"
)

type UserHandler struct {
	userUsecase domain.UserUsecase
}

func NewUserHandler(r *gin.RouterGroup, userUsecase domain.UserUsecase) {
	handler := &UserHandler{
		userUsecase: userUsecase,
	}

	userRoutes := r.Group("/users")
	{
		userRoutes.GET("", handler.GetAll)
		userRoutes.GET("/search", handler.Search)
		userRoutes.GET("/metrics", handler.GetMetrics)
		userRoutes.PUT("/:id/role", handler.UpdateRole)
		userRoutes.PUT("/:id/ban", handler.UpdateStatus)
	}
}

// GetAll godoc
// @Summary Ambil daftar semua pengguna (Khusus Admin)
// @Description Mengambil semua data pengguna dengan paginasi dan filter. Hanya bisa diakses oleh Admin.
// @Tags Users
// @Produce json
// @Security BearerAuth
// @Param page query int false "Nomor halaman (default: 1)"
// @Param limit query int false "Jumlah data per halaman (default: 10)"
// @Param search query string false "Pencarian nama atau email"
// @Param role_id query int false "Filter berdasarkan Role ID (1=Admin, 2=User, 3=Legal)"
// @Param is_banned query bool false "Filter berdasarkan status banned"
// @Param sort_by query string false "Kolom untuk sorting (username, email, created_at, role_id, is_banned)"
// @Param sort_dir query string false "Arah sorting (asc, desc)"
// @Success 200 {object} map[string]interface{} "Daftar pengguna dengan metadata paginasi"
// @Failure 401 {object} map[string]interface{} "Unauthorized"
// @Failure 403 {object} map[string]interface{} "Forbidden (Bukan Admin)"
// @Router /users [get]
func (h *UserHandler) GetAll(c *gin.Context) {
	role, exists := c.Get("role")
	if !exists || role.(string) != "Admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "only admin can access this"})
		return
	}

	page := 1
	limit := 10
	search := c.Query("search")

	if p := c.Query("page"); p != "" {
		_, _ = fmt.Sscanf(p, "%d", &page)
	}
	if l := c.Query("limit"); l != "" {
		_, _ = fmt.Sscanf(l, "%d", &limit)
	}

	var roleID *int
	if r := c.Query("role_id"); r != "" {
		var id int
		if _, err := fmt.Sscanf(r, "%d", &id); err == nil {
			roleID = &id
		}
	}

	var isBanned *bool
	if b := c.Query("is_banned"); b != "" {
		banned := b == "true"
		isBanned = &banned
	}

	sortBy := c.Query("sort_by")
	sortDir := c.Query("sort_dir")

	users, total, err := h.userUsecase.GetAllUsers(c.Request.Context(), page, limit, search, roleID, isBanned, sortBy, sortDir)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	lastPage := int((total + int64(limit) - 1) / int64(limit))
	if lastPage == 0 {
		lastPage = 1
	}
	hasNext := page < lastPage
	hasPrev := page > 1

	c.JSON(http.StatusOK, gin.H{
		"data": users,
		"meta": gin.H{
			"page":      page,
			"limit":     limit,
			"total":     total,
			"last_page": lastPage,
			"has_next":  hasNext,
			"has_prev":  hasPrev,
		},
	})
}

// Search godoc
// @Summary Cari pengguna untuk Autocomplete
// @Description Mencari pengguna berdasarkan username atau email
// @Tags Users
// @Produce json
// @Security BearerAuth
// @Param q query string false "Kata kunci pencarian"
// @Success 200 {array} map[string]interface{} "Daftar pengguna (id, username, email)"
// @Router /users/search [get]
func (h *UserHandler) Search(c *gin.Context) {
	search := c.Query("q")

	users, _, err := h.userUsecase.GetAllUsers(c.Request.Context(), 1, 20, search, nil, nil, "", "")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	var results []map[string]interface{}
	for _, u := range users {
		results = append(results, map[string]interface{}{
			"id":       u.ID,
			"username": u.Username,
			"email":    u.Email,
		})
	}

	c.JSON(http.StatusOK, results)
}

type UpdateRoleRequest struct {
	RoleID int `json:"role_id" binding:"required"`
}

// UpdateRole godoc
// @Summary Ubah Role pengguna (Khusus Admin)
// @Description Mengubah wewenang (Role) pengguna. Admin tidak bisa mengubah rolenya sendiri.
// @Tags Users
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "User ID"
// @Param request body v1.UpdateRoleRequest true "Payload Update Role"
// @Success 200 {object} map[string]interface{} "Berhasil diubah"
// @Failure 400 {object} map[string]interface{} "Bad Request"
// @Failure 401 {object} map[string]interface{} "Unauthorized"
// @Failure 403 {object} map[string]interface{} "Forbidden"
// @Router /users/{id}/role [put]
func (h *UserHandler) UpdateRole(c *gin.Context) {
	role, exists := c.Get("role")
	if !exists || role.(string) != "Admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "only admin can access this"})
		return
	}

	adminIDStr, _ := c.Get("user_id")
	adminID, _ := uuid.Parse(adminIDStr.(string))

	targetIDStr := c.Param("id")
	targetID, err := uuid.Parse(targetIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user ID"})
		return
	}

	var req UpdateRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.userUsecase.UpdateUserRole(c.Request.Context(), targetID, req.RoleID, adminID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "user role updated successfully"})
}

type UpdateStatusRequest struct {
	IsBanned bool `json:"is_banned"`
}

// UpdateStatus godoc
// @Summary Ban atau Unban pengguna (Khusus Admin)
// @Description Membanned atau melepas banned pengguna. Admin tidak bisa menge-ban dirinya sendiri.
// @Tags Users
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "User ID"
// @Param request body v1.UpdateStatusRequest true "Payload Update Status"
// @Success 200 {object} map[string]interface{} "Berhasil diubah"
// @Failure 400 {object} map[string]interface{} "Bad Request"
// @Failure 401 {object} map[string]interface{} "Unauthorized"
// @Failure 403 {object} map[string]interface{} "Forbidden"
// @Router /users/{id}/ban [put]
func (h *UserHandler) UpdateStatus(c *gin.Context) {
	role, exists := c.Get("role")
	if !exists || role.(string) != "Admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "only admin can access this"})
		return
	}

	adminIDStr, _ := c.Get("user_id")
	adminID, _ := uuid.Parse(adminIDStr.(string))

	targetIDStr := c.Param("id")
	targetID, err := uuid.Parse(targetIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user ID"})
		return
	}

	var req UpdateStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.userUsecase.UpdateUserStatus(c.Request.Context(), targetID, req.IsBanned, adminID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	statusMsg := "user unbanned successfully"
	if req.IsBanned {
		statusMsg = "user banned successfully"
	}

	c.JSON(http.StatusOK, gin.H{"message": statusMsg})
}

// GetMetrics godoc
// @Summary Dapatkan metrik pengguna
// @Description Mengambil total pengguna, pengguna aktif, pengguna di-ban, dan total admin
// @Tags Users
// @Produce json
// @Security BearerAuth
// @Success 200 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{} "Unauthorized"
// @Failure 403 {object} map[string]interface{} "Forbidden"
// @Failure 500 {object} map[string]interface{} "Internal Server Error"
// @Router /users/metrics [get]
func (h *UserHandler) GetMetrics(c *gin.Context) {
	role, exists := c.Get("role")
	if !exists || role.(string) != "Admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "only admin can access this"})
		return
	}

	metrics, err := h.userUsecase.GetUserMetrics(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data":   metrics,
	})
}
