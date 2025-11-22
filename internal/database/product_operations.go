package database

import (
	"context"
)

// CreateRegisteredProduct creates a new registered product
func (r *Repository) CreateRegisteredProduct(ctx context.Context, product *RegisteredProduct) error {
	query := `
		INSERT INTO registered_products (
			project_name, plan_name, stripe_product_id,
			stripe_price_monthly, stripe_price_yearly,
			monthly_amount, yearly_amount, currency,
			description, features, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, NOW(), NOW())
		RETURNING id, created_at, updated_at
	`

	return r.db.QueryRow(ctx, query,
		product.ProjectName,
		product.PlanName,
		product.StripeProductID,
		nullString(product.StripePriceMonthly),
		nullString(product.StripePriceYearly),
		nullInt64(product.MonthlyAmount),
		nullInt64(product.YearlyAmount),
		product.Currency,
		product.Description,
		product.Features,
	).Scan(&product.ID, &product.CreatedAt, &product.UpdatedAt)
}

// GetRegisteredProductsByProject retrieves all registered products for a project
func (r *Repository) GetRegisteredProductsByProject(ctx context.Context, projectName string) ([]*RegisteredProduct, error) {
	query := `
		SELECT id, project_name, plan_name, stripe_product_id,
			stripe_price_monthly, stripe_price_yearly,
			monthly_amount, yearly_amount, currency,
			description, features, created_at, updated_at
		FROM registered_products
		WHERE project_name = $1
		ORDER BY created_at DESC
	`

	rows, err := r.db.Query(ctx, query, projectName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var products []*RegisteredProduct
	for rows.Next() {
		product, err := ScanRegisteredProduct(rows)
		if err != nil {
			return nil, err
		}
		products = append(products, product)
	}

	return products, rows.Err()
}

// ProductExistsForProject checks if a product with the given plan name already exists for a project
// Returns (exists bool, stripeProductID string, error)
func (r *Repository) ProductExistsForProject(ctx context.Context, projectName, planName string) (bool, string, error) {
	var stripeProductID string
	query := `
		SELECT stripe_product_id FROM registered_products
		WHERE project_name = $1 AND plan_name = $2
		LIMIT 1
	`

	err := r.db.QueryRow(ctx, query, projectName, planName).Scan(&stripeProductID)
	if err != nil {
		if err.Error() == "no rows in result set" {
			return false, "", nil
		}
		return false, "", err
	}

	return true, stripeProductID, nil
}

// GetRegisteredProductByStripeID retrieves a registered product by its Stripe product ID
func (r *Repository) GetRegisteredProductByStripeID(ctx context.Context, stripeProductID string) (*RegisteredProduct, error) {
	query := `
		SELECT id, project_name, plan_name, stripe_product_id,
			stripe_price_monthly, stripe_price_yearly,
			monthly_amount, yearly_amount, currency,
			description, features, created_at, updated_at
		FROM registered_products
		WHERE stripe_product_id = $1
	`

	return ScanRegisteredProduct(r.db.QueryRow(ctx, query, stripeProductID))
}

// Helper functions for nullable fields
func nullString(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}

func nullInt64(i int64) interface{} {
	if i == 0 {
		return nil
	}
	return i
}
