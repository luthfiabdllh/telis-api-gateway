package http

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"telis-api-gateway/config"
	"telis-api-gateway/internal/delivery/http/middleware"
	v1 "telis-api-gateway/internal/delivery/http/v1"
	"telis-api-gateway/internal/domain"
	grpcClient "telis-api-gateway/internal/infrastructure/grpc"
	
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

func SetupRouter(cfg *config.Config, authUsecase domain.AuthUsecase, docUsecase domain.DocumentUsecase, redlineUsecase domain.RedlineUsecase, feedbackUsecase domain.FeedbackUsecase, metricsUsecase domain.MetricsUsecase, folderUsecase domain.FolderUsecase, agentClient grpcClient.AgentClient) *gin.Engine {
	r := gin.Default()

	// Health Check
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// Swagger UI Route
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// API V1 Group
	apiV1 := r.Group("/api/v1")
	{
		// Auth Routes (Public)
		v1.NewAuthHandler(apiV1, authUsecase)

		// Protected Routes
		protected := apiV1.Group("/")
		protected.Use(middleware.JWTAuthMiddleware(cfg))
		{
			// Document Routes
			v1.NewDocumentHandler(protected, docUsecase)

			// Folder Routes
			v1.NewFolderHandler(protected, folderUsecase)

			// Chat Routes
			v1.NewChatHandler(protected, agentClient)
			
			// Feedback Routes
			v1.NewFeedbackHandler(protected, feedbackUsecase)
			
			// Metrics Routes
			v1.NewMetricsHandler(protected, metricsUsecase)

			// Example Ping route accessible to everyone with a valid token
			protected.GET("/ping", func(c *gin.Context) {
				userID, _ := c.Get("user_id")
				role, _ := c.Get("role")
				c.JSON(http.StatusOK, gin.H{
					"message": "pong",
					"user_id": userID,
					"role":    role,
				})
			})

			// Admin Only Route Example
			adminGroup := protected.Group("/admin")
			adminGroup.Use(middleware.RoleMiddleware("Admin"))
			{
				adminGroup.GET("/stats", func(c *gin.Context) {
					c.JSON(http.StatusOK, gin.H{"message": "Secret Admin Stats"})
				})
			}

			// Admin and Legal Routes
			legalGroup := protected.Group("/")
			legalGroup.Use(middleware.RoleMiddleware("Admin", "Legal"))
			{
				v1.NewRedlineHandler(legalGroup, redlineUsecase)
			}
		}
	}

	return r
}
