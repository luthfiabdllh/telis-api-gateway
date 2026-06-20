package v1

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"telis-api-gateway/internal/domain"
	"telis-api-gateway/internal/delivery/http/middleware"
)

type MetricsHandler struct {
	metricsUsecase domain.MetricsUsecase
}

func NewMetricsHandler(r *gin.RouterGroup, metricsUsecase domain.MetricsUsecase) {
	handler := &MetricsHandler{
		metricsUsecase: metricsUsecase,
	}

	metricsRoutes := r.Group("/metrics")
	{
		// Only Admin can see global dashboard
		metricsRoutes.GET("/tokens", middleware.RoleMiddleware("admin"), handler.GetDashboardMetrics)
		// Any logged-in user can see their own metrics
		metricsRoutes.GET("/tokens/me", handler.GetMyMetrics)
	}
}

// GetDashboardMetrics godoc
// @Summary Dapatkan Dashboard Metrik Token (Khusus Admin)
// @Description Mengambil total biaya LLM bulan ini, top 5 pengguna, dan tren penggunaan harian.
// @Tags Metrics
// @Produce json
// @Security BearerAuth
// @Success 200 {object} domain.DashboardMetrics
// @Failure 401 {object} map[string]interface{} "Unauthorized"
// @Failure 403 {object} map[string]interface{} "Forbidden (Bukan Admin)"
// @Failure 500 {object} map[string]interface{} "Internal Server Error"
// @Router /metrics/tokens [get]
func (h *MetricsHandler) GetDashboardMetrics(c *gin.Context) {
	metrics, err := h.metricsUsecase.GetDashboardMetrics(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, metrics)
}

// GetMyMetrics godoc
// @Summary Dapatkan Metrik Token Saya Sendiri
// @Description Mengambil total biaya LLM yang telah dihabiskan oleh pengguna yang sedang login di bulan ini.
// @Tags Metrics
// @Produce json
// @Security BearerAuth
// @Success 200 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{} "Unauthorized"
// @Failure 500 {object} map[string]interface{} "Internal Server Error"
// @Router /metrics/tokens/me [get]
func (h *MetricsHandler) GetMyMetrics(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not found in token"})
		return
	}

	totalCost, err := h.metricsUsecase.GetMyMetrics(c.Request.Context(), userID.(string))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"user_id":               userID,
		"total_cost_this_month": totalCost,
	})
}
