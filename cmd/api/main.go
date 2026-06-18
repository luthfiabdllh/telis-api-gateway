package main

import (
	"log"

	"telis-api-gateway/config"
	"telis-api-gateway/internal/delivery/http"
	"telis-api-gateway/internal/repository"
	"telis-api-gateway/internal/usecase"
)

func main() {
	// 1. Load Configurations (.env)
	cfg := config.LoadConfig()

	// 2. Setup Database Connection
	db, err := config.ConnectDB(cfg)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	log.Println("Successfully connected to PostgreSQL")

	// 3. Dependency Injection (Wiring)
	// Repositories (Layer 3)
	userRepo := repository.NewUserRepository(db)

	// Usecases (Layer 2)
	authUsecase := usecase.NewAuthUsecase(userRepo, cfg)

	// 4. Setup Gin Router & Delivery Layer (Layer 4)
	router := http.SetupRouter(cfg, authUsecase)

	// 5. Start Server
	log.Printf("Starting API Gateway on port %s", cfg.Port)
	if err := router.Run(":" + cfg.Port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
