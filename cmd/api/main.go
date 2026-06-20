package main

import (
	"log"

	"telis-api-gateway/config"
	"telis-api-gateway/internal/delivery/http"
	grpcClient "telis-api-gateway/internal/infrastructure/grpc"
	"telis-api-gateway/internal/infrastructure/rabbitmq"
	"telis-api-gateway/internal/repository"
	"telis-api-gateway/internal/usecase"
	"telis-api-gateway/internal/domain"
	_ "telis-api-gateway/docs" // Swagger docs
)

// @title TELIS API Gateway
// @version 3.0
// @description Ini adalah dokumentasi API Gateway untuk proyek TELIS (Telkom Legal Intelligence System).
// @termsOfService http://swagger.io/terms/

// @contact.name API Support
// @contact.url http://www.swagger.io/support
// @contact.email support@swagger.io

// @license.name Apache 2.0
// @license.url http://www.apache.org/licenses/LICENSE-2.0.html

// @host localhost:8000
// @BasePath /api/v1

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization

func main() {
	// 1. Load Configurations (.env)
	cfg := config.LoadConfig()

	// 2. Setup Database Connection
	db, err := config.ConnectDB(cfg)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	
	// AutoMigrate database models
	err = db.AutoMigrate(
		&domain.UserFeedback{},
		&domain.ChatSession{},
		&domain.ChatMessage{},
	)
	if err != nil {
		log.Fatalf("Failed to auto-migrate database: %v", err)
	}
	
	log.Println("Successfully connected to PostgreSQL")

	// 3. Setup External Infrastructures (RabbitMQ & gRPC)
	rmqPublisher, err := rabbitmq.NewPublisher(cfg.RabbitMQURL)
	if err != nil {
		log.Fatalf("Failed to connect to RabbitMQ: %v", err)
	}
	defer rmqPublisher.Close()
	log.Println("Successfully connected to RabbitMQ")

	agentClient, err := grpcClient.NewAgentClient(cfg.AgentServiceURL)
	if err != nil {
		log.Fatalf("Failed to create Agent gRPC Client: %v", err)
	}
	defer agentClient.Close()
	log.Println("Successfully initialized gRPC Agent Client")

	// 4. Dependency Injection (Wiring)
	sqlDB, err := db.DB()
	if err != nil {
		log.Fatalf("Failed to get sql.DB: %v", err)
	}

	// Repositories (Layer 3)
	userRepo := repository.NewUserRepository(db)
	redlineRepo := repository.NewRedlineRepository(db)
	feedbackRepo := repository.NewFeedbackRepository(db)
	metricsRepo := repository.NewMetricsRepository(sqlDB)
	documentRepo := repository.NewDocumentRepository(sqlDB)
	folderRepo := repository.NewFolderRepository(sqlDB)
	chatRepo := repository.NewChatRepository(db)

	// Usecases (Layer 2)
	userUsecase := usecase.NewUserUsecase(userRepo)
	authUsecase := usecase.NewAuthUsecase(userRepo, cfg)
	feedbackUsecase := usecase.NewFeedbackUsecase(feedbackRepo)
	metricsUsecase := usecase.NewMetricsUsecase(metricsRepo)
	
	// Base dir for shared documents
	sharedDocsDir := "../shared_docs" // Assuming running from root of telis-api-gateway
	docUsecase := usecase.NewDocumentUsecase(rmqPublisher, documentRepo, sharedDocsDir)
	folderUsecase := usecase.NewFolderUsecase(folderRepo, docUsecase)
	redlineUsecase := usecase.NewRedlineUsecase(redlineRepo, rmqPublisher, sharedDocsDir)
	chatUsecase := usecase.NewChatUsecase(chatRepo)

	// 5. Setup Gin Router & Delivery Layer (Layer 4)
	router := http.SetupRouter(cfg, authUsecase, userUsecase, docUsecase, redlineUsecase, feedbackUsecase, metricsUsecase, folderUsecase, chatUsecase, agentClient)

	// 5. Start Server
	log.Printf("Starting API Gateway on port %s", cfg.Port)
	if err := router.Run(":" + cfg.Port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
