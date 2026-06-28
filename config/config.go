package config

import (
	"fmt"
	"log"
	"os"
	"strconv"

	"github.com/joho/godotenv"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type Config struct {
	Port                string
	DBHost              string
	DBUser              string
	DBPassword          string
	DBName              string
	DBPort              string
	JWTSecret           string
	JWTAccessExpMinutes int
	JWTRefreshExpDays   int
	InternalSSOSecret   string
	RabbitMQURL         string
	AgentServiceURL     string
	LegalEngineURL      string
	RedisURL            string
}

func LoadConfig() *Config {
	err := godotenv.Load()
	if err != nil {
		log.Println("No .env file found, relying on system environment variables")
	}

	accessExp, _ := strconv.Atoi(getEnv("JWT_ACCESS_EXP_MINUTES", "15"))
	refreshExp, _ := strconv.Atoi(getEnv("JWT_REFRESH_EXP_DAYS", "7"))

	return &Config{
		Port:                getEnv("PORT", "8000"),
		DBHost:              getEnv("DB_HOST", "localhost"),
		DBUser:              getEnv("DB_USER", "telis_admin"),
		DBPassword:          getEnv("DB_PASSWORD", "telis_secret_postgres"),
		DBName:              getEnv("DB_NAME", "telis_db"),
		DBPort:              getEnv("DB_PORT", "5432"),
		JWTSecret:           getEnv("JWT_SECRET", "secret"),
		JWTAccessExpMinutes: accessExp,
		JWTRefreshExpDays:   refreshExp,
		InternalSSOSecret:   getEnv("INTERNAL_SSO_SECRET", "secret"),
		RabbitMQURL:         getEnv("RABBITMQ_URL", "amqp://telis_rmq:telis_secret_rmq@localhost:5672/"),
		AgentServiceURL:     getEnv("AGENT_SERVICE_URL", "localhost:8001"),
		LegalEngineURL:      getEnv("LEGAL_ENGINE_GRPC_URL", "localhost:50054"),
		RedisURL:            getEnv("REDIS_URL", "redis://localhost:6379"),
	}
}

func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}

func ConnectDB(cfg *Config) (*gorm.DB, error) {
	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=disable TimeZone=Asia/Jakarta",
		cfg.DBHost, cfg.DBUser, cfg.DBPassword, cfg.DBName, cfg.DBPort)
	
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		return nil, err
	}
	return db, nil
}
