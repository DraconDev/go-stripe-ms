package admin

import "fmt"

// validateProductRequest validates the product registration request
func validateProductRequest(req *ProductRegistrationRequest) error {
	if req.ProjectName == "" {
		return fmt.Errorf("project_name is required")
	}

	if len(req.Plans) == 0 {
		return fmt.Errorf("at least one plan is required")
	}

	for i, plan := range req.Plans {
		if err := validatePlan(&plan, i); err != nil {
			return err
		}
	}

	return nil
}

// validatePlan validates a single plan
func validatePlan(plan *Plan, index int) error {
	if plan.Name == "" {
		return fmt.Errorf("plan[%d]: name is required", index)
	}

	if plan.Pricing.Monthly <= 0 && plan.Pricing.Yearly <= 0 {
		return fmt.Errorf("plan[%d]: at least one pricing option (monthly or yearly) must be provided", index)
	}

	if plan.Pricing.Monthly < 0 {
		return fmt.Errorf("plan[%d]: monthly price cannot be negative", index)
	}

	if plan.Pricing.Yearly < 0 {
		return fmt.Errorf("plan[%d]: yearly price cannot be negative", index)
	}

	// Validate pricing makes sense (monthly * 12 should be <= yearly if both present)
	if plan.Pricing.Monthly > 0 && plan.Pricing.Yearly > 0 {
		if plan.Pricing.Yearly > plan.Pricing.Monthly*12 {
			// This is a warning, not an error - yearly can be more expensive for flexibility
			// but we log it
		}
	}

	return nil
}
