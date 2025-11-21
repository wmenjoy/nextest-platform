# WebSocket Integration Guide

This guide shows how to integrate the WebSocket components into your `main.go` file.

## Integration Steps

### 1. Import Required Packages

Add these imports to your `main.go`:

```go
import (
    "test-management-service/internal/websocket"
    "test-management-service/internal/workflow"
    "test-management-service/internal/handler"
    "test-management-service/internal/repository"
    "test-management-service/internal/service"
)
```

### 2. Create and Start WebSocket Hub

After initializing your database, create the WebSocket hub and start it in a goroutine:

```go
func main() {
    // ... existing database initialization ...

    // Create WebSocket Hub
    hub := websocket.NewHub()

    // Start hub in background goroutine
    go hub.Run()

    // ... rest of initialization ...
}
```

### 3. Initialize Workflow Components

Create the workflow repositories and executor with the WebSocket hub:

```go
// Initialize workflow repositories
workflowRepo := repository.NewWorkflowRepository(db)
workflowRunRepo := repository.NewWorkflowRunRepository(db)
stepExecRepo := repository.NewStepExecutionRepository(db)
stepLogRepo := repository.NewStepLogRepository(db)
testCaseRepo := repository.NewWorkflowTestCaseRepository(db)

// Create test case repository (reuse if already exists)
caseRepo := repository.NewTestCaseRepository(db)

// Create unified executor (adjust target host as needed)
unifiedExecutor := testcase.NewUnifiedTestExecutor(
    cfg.Test.TargetHost,
    nil, // workflowExecutor - can be nil initially
    caseRepo,
    workflowRepo,
)

// Create workflow executor WITH WebSocket hub
workflowExecutor := workflow.NewWorkflowExecutor(
    db,
    testCaseRepo,
    workflowRepo,
    unifiedExecutor,
    hub, // Pass the hub to enable real-time broadcasting
)

// Update unified executor with workflow executor reference
unifiedExecutor.SetWorkflowExecutor(workflowExecutor)
```

### 4. Create Services

```go
// Create workflow service
workflowService := service.NewWorkflowService(
    workflowRepo,
    workflowRunRepo,
    stepExecRepo,
    stepLogRepo,
    testCaseRepo,
    workflowExecutor,
)
```

### 5. Register HTTP Handlers

```go
// Create handlers
workflowHandler := handler.NewWorkflowHandler(workflowService)
wsHandler := handler.NewWebSocketHandler(hub)

// Register routes
workflowHandler.RegisterRoutes(r)
wsHandler.RegisterRoutes(r)

log.Println("WebSocket workflow streaming enabled on /api/v2/workflows/runs/:runId/stream")
```

## Complete Example

Here's a complete example of what your `main.go` might look like:

```go
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
    "test-management-service/internal/websocket"
    "test-management-service/internal/workflow"

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
        &models.Workflow{},
        &models.WorkflowRun{},
        &models.WorkflowStepExecution{},
        &models.WorkflowStepLog{},
        &models.WorkflowVariableChange{},
    ); err != nil {
        log.Fatalf("Failed to migrate database: %v", err)
    }

    // ===== WebSocket Hub Setup =====
    hub := websocket.NewHub()
    go hub.Run()
    log.Println("WebSocket hub started")

    // ===== Repository Initialization =====
    caseRepo := repository.NewTestCaseRepository(db)
    groupRepo := repository.NewTestGroupRepository(db)
    resultRepo := repository.NewTestResultRepository(db)
    runRepo := repository.NewTestRunRepository(db)

    workflowRepo := repository.NewWorkflowRepository(db)
    workflowRunRepo := repository.NewWorkflowRunRepository(db)
    stepExecRepo := repository.NewStepExecutionRepository(db)
    stepLogRepo := repository.NewStepLogRepository(db)
    workflowTestCaseRepo := repository.NewWorkflowTestCaseRepository(db)

    // ===== Executor Initialization =====
    executor := testcase.NewExecutor(cfg.Test.TargetHost)

    // Create unified executor (without workflow executor initially)
    unifiedExecutor := testcase.NewUnifiedTestExecutor(
        cfg.Test.TargetHost,
        nil,
        caseRepo,
        workflowRepo,
    )

    // Create workflow executor WITH WebSocket hub
    workflowExecutor := workflow.NewWorkflowExecutor(
        db,
        workflowTestCaseRepo,
        workflowRepo,
        unifiedExecutor,
        hub, // Pass hub for real-time broadcasting
    )

    // Update unified executor with workflow executor reference
    unifiedExecutor.SetWorkflowExecutor(workflowExecutor)

    // ===== Service Initialization =====
    testService := service.NewTestService(caseRepo, groupRepo, resultRepo, runRepo, executor)

    workflowService := service.NewWorkflowService(
        workflowRepo,
        workflowRunRepo,
        stepExecRepo,
        stepLogRepo,
        workflowTestCaseRepo,
        workflowExecutor,
    )

    // ===== Handler Initialization =====
    testHandler := handler.NewTestHandler(testService)
    workflowHandler := handler.NewWorkflowHandler(workflowService)
    wsHandler := handler.NewWebSocketHandler(hub)

    // ===== Gin Router Setup =====
    r := gin.Default()

    // Enable CORS
    r.Use(corsMiddleware())

    // Register routes
    testHandler.RegisterRoutes(r)
    workflowHandler.RegisterRoutes(r)
    wsHandler.RegisterRoutes(r)

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
    log.Printf("WebSocket streaming available at: ws://%s/api/v2/workflows/runs/:runId/stream", addr)
    if err := r.Run(addr); err != nil {
        log.Fatalf("Failed to start server: %v", err)
    }
}

func initDatabase(cfg *config.Config) (*gorm.DB, error) {
    switch cfg.Database.Type {
    case "sqlite":
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
```

## Key Points

1. **Hub Must Run**: Always call `go hub.Run()` after creating the hub. This starts the message distribution loop.

2. **Pass Hub to Executor**: The workflow executor needs the hub reference to broadcast events. Pass it as the last parameter to `NewWorkflowExecutor`.

3. **Register Routes**: Both workflow handler and WebSocket handler must be registered with the Gin router.

4. **Migration**: Don't forget to add workflow models to your `AutoMigrate` call:
   - `models.Workflow`
   - `models.WorkflowRun`
   - `models.WorkflowStepExecution`
   - `models.WorkflowStepLog`
   - `models.WorkflowVariableChange`

## Testing the Integration

After starting your server, test the WebSocket connection:

```bash
# Start a workflow
curl -X POST http://localhost:8080/api/v2/workflows/my-workflow/execute

# Connect to WebSocket with the returned runId
wscat -c "ws://localhost:8080/api/v2/workflows/runs/<RUN_ID>/stream"
```

You should see real-time messages as the workflow executes!

## Troubleshooting

### No Messages Received

- Verify `hub.Run()` is called in a goroutine
- Check that the hub is passed to `NewWorkflowExecutor`
- Ensure the runId exists and is valid

### Connection Refused

- Verify WebSocket routes are registered
- Check CORS settings if connecting from browser
- Ensure server is running on expected port

### Build Errors

- Run `go mod tidy` to update dependencies
- Verify all imports are correct
- Check that gorilla/websocket is installed: `go get github.com/gorilla/websocket`
