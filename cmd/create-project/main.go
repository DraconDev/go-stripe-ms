package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/DraconDev/go-stripe-ms/internal/database"
	"github.com/jackc/pgx/v5"
	"github.com/joho/godotenv"
)

func main() {
	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using environment variables")
	}

	// Get database URL
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatal("DATABASE_URL environment variable is required")
	}

	// Connect to database
	ctx := context.Background()
	conn, err := pgx.Connect(ctx, dbURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer conn.Close(ctx)

	// Create repository
	repo := database.NewRepository(conn)

	// Initialize tables (will create projects table)
	if err := repo.InitializeTables(ctx); err != nil {
		log.Fatalf("Failed to initialize tables: %v", err)
	}

	// Get project name from command line or use default
	projectName := "Default Project"
	if len(os.Args) > 1 {
		projectName = os.Args[1]
	}

	// Create a new project
	project, err := repo.CreateProject(ctx, projectName, "")
	if err != nil {
		log.Fatalf("Failed to create project: %v", err)
	}

	// Display the project details
	fmt.Println("✅ Project created successfully!")
	fmt.Println()
	fmt.Println("Project Details:")
	fmt.Printf("  ID:         %s\n", project.ID)
	fmt.Printf("  Name:       %s\n", project.Name)
	fmt.Printf("  API Key:    %s\n", project.APIKey)
	fmt.Printf("  Is Active:  %v\n", project.IsActive)
	fmt.Println()
	fmt.Println("🔑 Save this API key! You'll need it to authenticate requests.")
	fmt.Println()
	fmt.Println("Example usage:")
	fmt.Printf("  curl -H \"X-API-Key: %s\" http://localhost:9000/health\n", project.APIKey)
	fmt.Println()
}
