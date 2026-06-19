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
)

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
	// Repositories (Layer 3)
	userRepo := repository.NewUserRepository(db)
	redlineRepo := repository.NewRedlineRepository(db)
	feedbackRepo := repository.NewFeedbackRepository(db)

	// Usecases (Layer 2)
	authUsecase := usecase.NewAuthUsecase(userRepo, cfg)
	feedbackUsecase := usecase.NewFeedbackUsecase(feedbackRepo)
	
	// Base dir for shared documents
	sharedDocsDir := "../shared_docs" // Assuming running from root of telis-api-gateway
	docUsecase := usecase.NewDocumentUsecase(rmqPublisher, sharedDocsDir)
	redlineUsecase := usecase.NewRedlineUsecase(redlineRepo, rmqPublisher, sharedDocsDir)

	// 5. Setup Gin Router & Delivery Layer (Layer 4)
	router := http.SetupRouter(cfg, authUsecase, docUsecase, redlineUsecase, feedbackUsecase, agentClient)

	// 5. Start Server
	log.Printf("Starting API Gateway on port %s", cfg.Port)
	if err := router.Run(":" + cfg.Port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
