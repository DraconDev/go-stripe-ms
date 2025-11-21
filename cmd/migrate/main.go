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

	fmt.Println("🔄 Starting database migration...")
	fmt.Println()

	// Step 1: Create projects table
	fmt.Println("1. Creating projects table...")
	_, err = conn.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS projects (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			name VARCHAR(255) NOT NULL,
			api_key VARCHAR(64) UNIQUE NOT NULL,
			webhook_url TEXT,
			is_active BOOLEAN DEFAULT true,
			created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
			updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
		)
	`)
	if err != nil {
		log.Fatalf("Failed to create projects table: %v", err)
	}
	fmt.Println("   ✅ Projects table created")

	// Step 2: Create default project
	fmt.Println("2. Creating default project...")
	repo := database.NewRepository(conn)

	// Check if default project already exists
	var projectCount int
	err = conn.QueryRow(ctx, "SELECT COUNT(*) FROM projects").Scan(&projectCount)
	if err != nil {
		log.Fatalf("Failed to check projects: %v", err)
	}

	var defaultProjectID string
	var apiKey string

	if projectCount == 0 {
		project, err := repo.CreateProject(ctx, "Default Project", "")
		if err != nil {
			log.Fatalf("Failed to create default project: %v", err)
		}
		defaultProjectID = project.ID.String()
		apiKey = project.APIKey
		fmt.Printf("   ✅ Default project created (ID: %s)\n", defaultProjectID)
	} else {
		// Get existing project
		var id, key string
		err = conn.QueryRow(ctx, "SELECT id, api_key FROM projects LIMIT 1").Scan(&id, &key)
		if err != nil {
			log.Fatalf("Failed to get existing project: %v", err)
		}
		defaultProjectID = id
		apiKey = key
		fmt.Printf("   ℹ️  Using existing project (ID: %s)\n", defaultProjectID)
	}

	// Step 3: Add project_id to customers table (if not exists)
	fmt.Println("3. Adding project_id to customers table...")
	_, err = conn.Exec(ctx, `
		ALTER TABLE customers 
		ADD COLUMN IF NOT EXISTS project_id UUID REFERENCES projects(id) ON DELETE CASCADE
	`)
	if err != nil {
		log.Fatalf("Failed to add project_id to customers: %v", err)
	}
	fmt.Println("   ✅ project_id column added to customers")

	// Step 4: Update existing customers with default project
	fmt.Println("4. Updating existing customers...")
	result, err := conn.Exec(ctx, `
		UPDATE customers 
		SET project_id = $1 
		WHERE project_id IS NULL
	`, defaultProjectID)
	if err != nil {
		log.Fatalf("Failed to update customers: %v", err)
	}
	rowsAffected := result.RowsAffected()
	fmt.Printf("   ✅ Updated %d customer records\n", rowsAffected)

	// Step 5: Add project_id to subscriptions table (if not exists)
	fmt.Println("5. Adding project_id to subscriptions table...")
	_, err = conn.Exec(ctx, `
		ALTER TABLE subscriptions 
		ADD COLUMN IF NOT EXISTS project_id UUID REFERENCES projects(id) ON DELETE CASCADE
	`)
	if err != nil {
		log.Fatalf("Failed to add project_id to subscriptions: %v", err)
	}
	fmt.Println("   ✅ project_id column added to subscriptions")

	// Step 6: Update existing subscriptions with default project
	fmt.Println("6. Updating existing subscriptions...")
	result, err = conn.Exec(ctx, `
		UPDATE subscriptions 
		SET project_id = $1 
		WHERE project_id IS NULL
	`, defaultProjectID)
	if err != nil {
		log.Fatalf("Failed to update subscriptions: %v", err)
	}
	rowsAffected = result.RowsAffected()
	fmt.Printf("   ✅ Updated %d subscription records\n", rowsAffected)

	// Step 7: Drop old unique constraints and create new ones
	fmt.Println("7. Updating constraints...")

	// Drop old constraint on customers
	_, _ = conn.Exec(ctx, `ALTER TABLE customers DROP CONSTRAINT IF EXISTS customers_user_id_key`)
	_, err = conn.Exec(ctx, `
		CREATE UNIQUE INDEX IF NOT EXISTS customers_project_user_unique 
		ON customers(project_id, user_id)
	`)
	if err != nil {
		log.Printf("   ⚠️  Warning: Could not create unique index on customers: %v", err)
	} else {
		fmt.Println("   ✅ Updated customers unique constraint")
	}

	// Drop old constraint on subscriptions
	_, _ = conn.Exec(ctx, `DROP INDEX IF EXISTS subscriptions_user_id_product_id_key`)
	_, err = conn.Exec(ctx, `
		CREATE UNIQUE INDEX IF NOT EXISTS subscriptions_project_user_product_unique 
		ON subscriptions(project_id, user_id, product_id)
	`)
	if err != nil {
		log.Printf("   ⚠️  Warning: Could not create unique index on subscriptions: %v", err)
	} else {
		fmt.Println("   ✅ Updated subscriptions unique constraint")
	}

	// Step 8: Create indexes
	fmt.Println("8. Creating indexes...")
	indexes := []string{
		`CREATE INDEX IF NOT EXISTS idx_projects_api_key ON projects(api_key)`,
		`CREATE INDEX IF NOT EXISTS idx_customers_project_id ON customers(project_id)`,
		`CREATE INDEX IF NOT EXISTS idx_subscriptions_project_id ON subscriptions(project_id)`,
	}

	for _, idx := range indexes {
		_, err = conn.Exec(ctx, idx)
		if err != nil {
			log.Printf("   ⚠️  Warning: Could not create index: %v", err)
		}
	}
	fmt.Println("   ✅ Indexes created")

	// Done!
	fmt.Println()
	fmt.Println("✅ Migration completed successfully!")
	fmt.Println()
	fmt.Println("═══════════════════════════════════════════════════════")
	fmt.Println("🔑 YOUR API KEY (save this!):")
	fmt.Println()
	fmt.Printf("   %s\n", apiKey)
	fmt.Println()
	fmt.Println("═══════════════════════════════════════════════════════")
	fmt.Println()
	fmt.Println("Example usage:")
	fmt.Printf("  curl -H \"X-API-Key: %s\" http://localhost:9000/health\n", apiKey)
	fmt.Println()
	fmt.Println("💡 Tip: You can use this same API key for all your projects!")
	fmt.Println()
}
