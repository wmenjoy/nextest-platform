package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"

	"test-management-service/internal/config"
	"test-management-service/internal/models"
	"test-management-service/internal/repository"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// GroupData represents a test group from JSON
type GroupData struct {
	GroupID     string `json:"groupId"`
	Name        string `json:"name"`
	ParentID    string `json:"parentId"`
	Description string `json:"description"`
}

// TestCaseData represents a test case from JSON
type TestCaseData struct {
	TestID      string                 `json:"testId"`
	GroupID     string                 `json:"groupId"`
	Name        string                 `json:"name"`
	Type        string                 `json:"type"`
	Priority    string                 `json:"priority"`
	Objective   string                 `json:"objective"`
	Timeout     int                    `json:"timeout"`
	HTTP        map[string]interface{} `json:"http"`
	Command     map[string]interface{} `json:"command"`
	Assertions  []interface{}          `json:"assertions"`
	Tags        []interface{}          `json:"tags"`
}

func main() {
	configPath := flag.String("config", "config.toml", "Path to config file")
	dataPath := flag.String("data", "examples/sample-tests.json", "Path to test data JSON file")
	flag.Parse()

	// Load configuration
	cfg, err := config.LoadConfig(*configPath)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Initialize database
	db, err := initDatabase(cfg)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}

	// Auto migrate
	if err := db.AutoMigrate(
		&models.TestGroup{},
		&models.TestCase{},
		&models.TestResult{},
		&models.TestRun{},
	); err != nil {
		log.Fatalf("Failed to migrate database: %v", err)
	}

	// Initialize repositories
	groupRepo := repository.NewTestGroupRepository(db)
	caseRepo := repository.NewTestCaseRepository(db)

	// Read test data
	data, err := os.ReadFile(*dataPath)
	if err != nil {
		log.Fatalf("Failed to read data file: %v", err)
	}

	var testData struct {
		Groups []GroupData     `json:"groups"`
		Tests  []TestCaseData  `json:"tests"`
	}

	if err := json.Unmarshal(data, &testData); err != nil {
		log.Fatalf("Failed to parse JSON: %v", err)
	}

	// Import groups
	fmt.Println("Importing test groups...")
	for _, g := range testData.Groups {
		group := &models.TestGroup{
			GroupID:     g.GroupID,
			Name:        g.Name,
			ParentID:    g.ParentID,
			Description: g.Description,
		}

		// Check if exists
		existing, _ := groupRepo.FindByID(g.GroupID)
		if existing != nil {
			fmt.Printf("  ⏭  Group '%s' already exists, skipping\n", g.Name)
			continue
		}

		if err := groupRepo.Create(group); err != nil {
			log.Printf("  ❌ Failed to create group '%s': %v\n", g.Name, err)
		} else {
			fmt.Printf("  ✓ Created group: %s\n", g.Name)
		}
	}

	// Import tests
	fmt.Println("\nImporting test cases...")
	for _, t := range testData.Tests {
		testCase := &models.TestCase{
			TestID:    t.TestID,
			GroupID:   t.GroupID,
			Name:      t.Name,
			Type:      t.Type,
			Priority:  t.Priority,
			Status:    "active",
			Objective: t.Objective,
			Timeout:   t.Timeout,
		}

		if t.HTTP != nil {
			testCase.HTTPConfig = t.HTTP
		}
		if t.Command != nil {
			testCase.CommandConfig = t.Command
		}
		if t.Assertions != nil {
			testCase.Assertions = t.Assertions
		}
		if t.Tags != nil {
			testCase.Tags = t.Tags
		}

		// Check if exists
		existing, _ := caseRepo.FindByID(t.TestID)
		if existing != nil {
			fmt.Printf("  ⏭  Test '%s' already exists, skipping\n", t.Name)
			continue
		}

		if err := caseRepo.Create(testCase); err != nil {
			log.Printf("  ❌ Failed to create test '%s': %v\n", t.Name, err)
		} else {
			fmt.Printf("  ✓ Created test: %s\n", t.Name)
		}
	}

	fmt.Println("\n✅ Import completed!")
}

func initDatabase(cfg *config.Config) (*gorm.DB, error) {
	switch cfg.Database.Type {
	case "sqlite":
		db, err := gorm.Open(sqlite.Open(cfg.Database.DSN), &gorm.Config{})
		if err != nil {
			return nil, fmt.Errorf("failed to open sqlite database: %w", err)
		}
		return db, nil
	default:
		return nil, fmt.Errorf("unsupported database type: %s", cfg.Database.Type)
	}
}
