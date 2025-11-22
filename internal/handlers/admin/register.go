package admin

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/DraconDev/go-stripe-ms/internal/database"
	"github.com/DraconDev/go-stripe-ms/internal/handlers/utils"
	"github.com/stripe/stripe-go/v72"
	"github.com/stripe/stripe-go/v72/price"
	"github.com/stripe/stripe-go/v72/product"
)

// HandleProductRegistration handles the POST /admin/products/register endpoint
func HandleProductRegistration(db database.RepositoryInterface, stripeSecret string, w http.ResponseWriter, r *http.Request) {
	stripe.Key = stripeSecret
	ctx := r.Context()

	// Parse request
	var req ProductRegistrationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.WriteErrorResponse(w, http.StatusBadRequest, "INVALID_JSON", "Failed to parse request body", err.Error())
		return
	}

	// Validate request
	if err := validateProductRequest(&req); err != nil {
		utils.WriteErrorResponse(w, http.StatusBadRequest, "VALIDATION_ERROR", err.Error(), "")
		return
	}

	// Check if products already exist for this project
	for _, plan := range req.Plans {
		exists, existingProductID, err := db.ProductExistsForProject(ctx, req.ProjectName, plan.Name)
		if err != nil {
			utils.WriteErrorResponse(w, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to check existing products", err.Error())
			return
		}
		if exists {
			respondWithConflict(w, req.ProjectName, plan.Name, existingProductID)
			return
		}
	}

	// Create products in Stripe
	products, err := createStripeProducts(ctx, req)
	if err != nil {
		log.Printf("Failed to create Stripe products: %v", err)
		utils.WriteErrorResponse(w, http.StatusInternalServerError, "STRIPE_ERROR", "Failed to create products in Stripe", err.Error())
		return
	}

	// Store products in database
	if err := storeProducts(ctx, db, req.ProjectName, products); err != nil {
		log.Printf("Failed to store products in database: %v", err)
		// Rollback Stripe products
		rollbackStripeProducts(products)
		utils.WriteErrorResponse(w, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to store products", err.Error())
		return
	}

	// Return success response
	response := RegistrationResponse{
		Success:   true,
		ProjectID: req.ProjectName,
		Products:  products,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
	log.Printf("Successfully registered %d products for project '%s'", len(products), req.ProjectName)
}

// createStripeProducts creates products and prices in Stripe
func createStripeProducts(ctx context.Context, req ProductRegistrationRequest) ([]ProductResponse, error) {
	var results []ProductResponse

	for _, plan := range req.Plans {
		// Create Stripe Product
		productParams := &stripe.Product

		Params{
			Name:        stripe.String(fmt.Sprintf("%s - %s", req.ProjectName, plan.Name)),
			Description: stripe.String(plan.Description),
		}

		// Add metadata
		productParams.Metadata = map[string]string{
			"project_name": req.ProjectName,
			"plan_name":    plan.Name,
		}
		if len(plan.Features) > 0 {
			productParams.Metadata["features"] = strings.Join(plan.Features, ",")
		}

		stripeProduct, err := product.New(productParams)
		if err != nil {
			return nil, fmt.Errorf("failed to create product for plan '%s': %w", plan.Name, err)
		}

		log.Printf("Created Stripe product: %s for plan '%s'", stripeProduct.ID, plan.Name)

		// Create prices
		prices := PriceResponse{}

		// Monthly price
		if plan.Pricing.Monthly > 0 {
			monthlyPrice, err := price.New(&stripe.PriceParams{
				Product:    stripe.String(stripeProduct.ID),
				UnitAmount: stripe.Int64(plan.Pricing.Monthly),
				Currency:   stripe.String("usd"),
				Recurring: &stripe.PriceRecurringParams{
					Interval: stripe.String("month"),
				},
				Metadata: map[string]string{
					"project_name": req.ProjectName,
					"plan_name":    plan.Name,
					"interval":     "monthly",
				},
			})
			if err != nil {
				return nil, fmt.Errorf("failed to create monthly price for plan '%s': %w", plan.Name, err)
			}
			prices.Monthly = &PriceDetails{
				StripePriceID: monthlyPrice.ID,
				Amount:        monthlyPrice.UnitAmount,
				Interval:      "month",
				Currency:      "usd",
			}
			log.Printf("Created monthly price: %s for plan '%s'", monthlyPrice.ID, plan.Name)
		}

		// Yearly price (if specified)
		if plan.Pricing.Yearly > 0 {
			yearlyPrice, err := price.New(&stripe.PriceParams{
				Product:    stripe.String(stripeProduct.ID),
				UnitAmount: stripe.Int64(plan.Pricing.Yearly),
				Currency:   stripe.String("usd"),
				Recurring: &stripe.PriceRecurringParams{
					Interval: stripe.String("year"),
				},
				Metadata: map[string]string{
					"project_name": req.ProjectName,
					"plan_name":    plan.Name,
					"interval":     "yearly",
				},
			})
			if err != nil {
				return nil, fmt.Errorf("failed to create yearly price for plan '%s': %w", plan.Name, err)
			}
			prices.Yearly = &PriceDetails{
				StripePriceID: yearlyPrice.ID,
				Amount:        yearlyPrice.UnitAmount,
				Interval:      "year",
				Currency:      "usd",
			}
			log.Printf("Created yearly price: %s for plan '%s'", yearlyPrice.ID, plan.Name)
		}

		results = append(results, ProductResponse{
			PlanName:        plan.Name,
			StripeProductID: stripeProduct.ID,
			Prices:          prices,
			CreatedAt:       time.Now(),
		})
	}

	return results, nil
}

// storeProducts persists the created products to the database
func storeProducts(ctx context.Context, db database.RepositoryInterface, projectName string, products []ProductResponse) error {
	for _, product := range products {
		// Convert features to JSON if needed (we'll store empty for now)
		featuresJSON := []byte("{}")

		dbProduct := &database.RegisteredProduct{
			ProjectName:        projectName,
			PlanName:           product.PlanName,
			StripeProductID:    product.StripeProductID,
			StripePriceMonthly: getMonthlyPriceID(product.Prices),
			StripePriceYearly:  getYearlyPriceID(product.Prices),
			MonthlyAmount:      getMonthlyAmount(product.Prices),
			YearlyAmount:       getYearlyAmount(product.Prices),
			Currency:           "usd",
			Features:           featuresJSON,
		}

		if err := db.CreateRegisteredProduct(ctx, dbProduct); err != nil {
			return fmt.Errorf("failed to create product record for '%s': %w", product.PlanName, err)
		}
		log.Printf("Stored product '%s' in database", product.PlanName)
	}

	return nil
}

// rollbackStripeProducts archives products in Stripe if database save fails
func rollbackStripeProducts(products []ProductResponse) {
	log.Printf("Rolling back %d Stripe products", len(products))
	for _, prod := range products {
		_, err := product.Update(prod.StripeProductID, &stripe.ProductParams{
			Active: stripe.Bool(false),
		})
		if err != nil {
			log.Printf("Failed to archive product %s during rollback: %v", prod.StripeProductID, err)
		} else {
			log.Printf("Archived product %s", prod.StripeProductID)
		}
	}
}

// Helper functions to extract price details
func getMonthlyPriceID(prices PriceResponse) string {
	if prices.Monthly != nil {
		return prices.Monthly.StripePriceID
	}
	return ""
}

func getYearlyPriceID(prices PriceResponse) string {
	if prices.Yearly != nil {
		return prices.Yearly.StripePriceID
	}
	return ""
}

func getMonthlyAmount(prices PriceResponse) int64 {
	if prices.Monthly != nil {
		return prices.Monthly.Amount
	}
	return 0
}

func getYearlyAmount(prices PriceResponse) int64 {
	if prices.Yearly != nil {
		return prices.Yearly.Amount
	}
	return 0
}

// respondWithConflict sends a 409 Conflict response
func respondWithConflict(w http.ResponseWriter, projectName, planName, existingProductID string) {
	response := ErrorResponse{
		Success:           false,
		Error:             "ALREADY_EXISTS",
		Message:           fmt.Sprintf("Product for project '%s' plan '%s' already exists", projectName, planName),
		ExistingProductID: existingProductID,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusConflict)
	json.NewEncoder(w).Encode(response)
}
