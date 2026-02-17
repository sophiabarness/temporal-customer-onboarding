package activities

import (
	"context"
	"fmt"
	"math/rand"
	"unicode"

	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/temporal"

	"temporal-customer-onboarding/shared"
)

// ValidateWithSupplier sends the merchant's identity document to a
// third-party verification supplier (e.g., Onfido, Jumio) and returns the result.
// Idempotency: naturally idempotent — validation is a read operation with no side effects.
func (a *Activities) ValidateWithSupplier(ctx context.Context, doc shared.DocumentUpload) (shared.VerificationResult, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("Sending document to verification supplier",
		"merchantId", doc.MerchantID,
		"documentType", doc.DocumentType,
		"documentId", doc.DocumentID,
	)

	// Validate: document ID must contain only digits.
	// In production, the supplier would perform this validation.
	// Validate: document ID must contain only digits.
	// In production, the supplier would perform this validation.
	for _, ch := range doc.DocumentID {
		if !unicode.IsDigit(ch) {
			logger.Info("Supplier rejected identity document — contains non-numeric characters",
				"merchantId", doc.MerchantID,
				"documentId", doc.DocumentID,
			)
			return shared.VerificationResult{
					Passed:         false,
					VerificationID: fmt.Sprintf("SUP-FAIL-%s", doc.MerchantID),
					Details:        fmt.Sprintf("Document ID '%s' contains non-numeric characters", doc.DocumentID),
				}, temporal.NewNonRetryableApplicationError(
					"identity document rejected by supplier: contains non-numeric characters",
					shared.ErrTypeIdentityVerificationFailed,
					nil,
				)
		}
	}

	// -------------------------------------------------------------------------
	// DEMO: Simulating a flaky 3rd party API for the "Fault Tolerance" demo.
	// If the document ID ends with "99", we simulate a failure.
	// -------------------------------------------------------------------------
	// -------------------------------------------------------------------------
	// DEMO: Simulating a flaky 3rd party API for the "Fault Tolerance" demo.
	// 75% chance of failure to demonstrate automatic activity retries.
	// -------------------------------------------------------------------------
	if rand.Float64() < 0.75 {
		logger.Error("Simulating 3rd party API downtime (75% chance)",
			"merchantId", doc.MerchantID,
		)
		return shared.VerificationResult{}, fmt.Errorf("simulated 3rd party API failure (transient)")
	}

	verificationID := fmt.Sprintf("SUP-%s", doc.MerchantID)
	logger.Info("Supplier verified document successfully", "verificationId", verificationID)

	return shared.VerificationResult{
		Passed:         true,
		VerificationID: verificationID,
		Details:        "Identity document verified by supplier",
	}, nil
}

// PerformInternalVerifications runs internal identity verification:
// checks whether the supplier-verified identity matches the merchant account.
//
// Idempotency: naturally idempotent — validation is a read operation with no side effects.
func (a *Activities) PerformInternalVerifications(ctx context.Context, merchantID string) (shared.VerificationResult, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("Performing internal identity verification", "merchantId", merchantID)

	// In production: verify that the supplier-verified identity matches merchant records.
	verificationID := fmt.Sprintf("INT-%s", merchantID)
	logger.Info("Internal identity verification passed", "verificationId", verificationID)

	return shared.VerificationResult{
		Passed:         true,
		VerificationID: verificationID,
		Details:        "Internal identity verification passed",
	}, nil
}
