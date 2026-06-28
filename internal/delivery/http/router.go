package http

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"telis-api-gateway/config"
	"telis-api-gateway/internal/delivery/http/middleware"
	v1 "telis-api-gateway/internal/delivery/http/v1"
	"telis-api-gateway/internal/domain"
	grpcClient "telis-api-gateway/internal/infrastructure/grpc"
	
	"github.com/go-redis/redis_rate/v10"
	"github.com/redis/go-redis/v9"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"github.com/gin-contrib/cors"
)

func SetupRouter(cfg *config.Config, authUsecase domain.AuthUsecase, userUsecase domain.UserUsecase, docUsecase domain.DocumentUsecase, redlineUsecase domain.RedlineUsecase, feedbackUsecase domain.FeedbackUsecase, metricsUsecase domain.MetricsUsecase, folderUsecase domain.FolderUsecase, chatUsecase domain.ChatUsecase, agentClient grpcClient.AgentClient, redisClient *redis.Client) *gin.Engine {
	r := gin.Default()

	// Enable CORS
	configCORS := cors.DefaultConfig()
	configCORS.AllowAllOrigins = true
	configCORS.AllowCredentials = true
	configCORS.AllowHeaders = []string{"Origin", "Content-Length", "Content-Type", "Authorization", "Accept"}
	r.Use(cors.New(configCORS))

	// Global Middlewares
	r.Use(middleware.CorrelationIDMiddleware())

	// Health Check
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// Swagger UI Route
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// API V1 Group
	apiV1 := r.Group("/api/v1")
	{
		// Protected Routes
		protected := apiV1.Group("")
		protected.Use(middleware.JWTAuthMiddleware(cfg))
		
		// Apply Rate Limiter to /chat/stream
		// Example: 20 requests per minute
		chatStreamLimit := redis_rate.PerMinute(20)
		rateLimiter := middleware.RateLimiterMiddleware(redisClient, chatStreamLimit)

		// Auth Routes (Public & Protected)
		v1.NewAuthHandler(apiV1, protected, authUsecase, cfg.InternalSSOSecret)

		{
			// Document Routes
			v1.NewDocumentHandler(protected, docUsecase)

			// User Routes
			v1.NewUserHandler(protected, userUsecase)

			// Folder Routes
			v1.NewFolderHandler(protected, folderUsecase)

			// Chat Routes
			chatGroup := protected.Group("")
			chatGroup.Use(func(c *gin.Context) {
				if c.Request.URL.Path == "/api/v1/chat/stream" {
					rateLimiter(c)
				} else {
					c.Next()
				}
			})
			v1.NewChatHandler(chatGroup, agentClient, chatUsecase)
			
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
