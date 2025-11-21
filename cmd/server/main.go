package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"test-management-service/internal/config"
	"test-management-service/internal/handler"
	"test-management-service/internal/models"
	"test-management-service/internal/repository"
	"test-management-service/internal/service"
	"test-management-service/internal/testcase"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func main() {
	// Load configuration
	configPath := os.Getenv("CONFIG_FILE")
	if configPath == "" {
		configPath = "config.toml"
	}

	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Initialize database
	db, err := initDatabase(cfg)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}

	// Auto migrate models
	if err := db.AutoMigrate(
		&models.TestGroup{},
		&models.TestCase{},
		&models.TestResult{},
		&models.TestRun{},
		&models.Environment{},
		&models.EnvironmentVariable{},
	); err != nil {
		log.Fatalf("Failed to migrate database: %v", err)
	}

	// Initialize repositories
	caseRepo := repository.NewTestCaseRepository(db)
	groupRepo := repository.NewTestGroupRepository(db)
	resultRepo := repository.NewTestResultRepository(db)
	runRepo := repository.NewTestRunRepository(db)
	envRepo := repository.NewEnvironmentRepository(db)
	envVarRepo := repository.NewEnvironmentVariableRepository(db)

	// Initialize environment service and variable injector
	envService := service.NewEnvironmentService(envRepo, envVarRepo)
	variableInjector := service.NewVariableInjector(envService)

	// Initialize executor with variable injection
	executor := testcase.NewExecutorWithInjector(cfg.Test.TargetHost, nil, caseRepo, nil, variableInjector)

	// Initialize service
	testService := service.NewTestService(caseRepo, groupRepo, resultRepo, runRepo, executor)

	// Initialize handlers
	testHandler := handler.NewTestHandler(testService)
	envHandler := handler.NewEnvironmentHandler(envService)

	// Setup Gin router
	r := gin.Default()

	// Enable CORS
	r.Use(corsMiddleware())

	// Register routes
	testHandler.RegisterRoutes(r)
	envHandler.RegisterRoutes(r)

	// Serve static files (Web UI)
	r.Static("/web", "./web")
	r.GET("/", func(c *gin.Context) {
		c.Redirect(302, "/web/app.html")
	})

	// Health check endpoint
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status":  "ok",
			"service": "test-management-service",
		})
	})

	// Start server
	addr := cfg.Server.GetAddr()
	log.Printf("Starting test management service on %s", addr)
	if err := r.Run(addr); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

func initDatabase(cfg *config.Config) (*gorm.DB, error) {
	switch cfg.Database.Type {
	case "sqlite":
		// Ensure data directory exists
		dbPath := cfg.Database.DSN
		dbDir := filepath.Dir(dbPath)
		if err := os.MkdirAll(dbDir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create data directory: %w", err)
		}

		db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
		if err != nil {
			return nil, fmt.Errorf("failed to open sqlite database: %w", err)
		}
		return db, nil

	default:
		return nil, fmt.Errorf("unsupported database type: %s", cfg.Database.Type)
	}
}

func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}
