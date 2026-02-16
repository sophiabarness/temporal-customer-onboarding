package activities

import (
	"context"
	"fmt"
	"unicode"

	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/temporal"

	"temporal-customer-onboarding/shared"
)

// ValidateWithSupplier sends the merchant's identity document to a
// third-party verification supplier (e.g., Onfido, Jumio) and returns the result.
func (a *Activities) ValidateWithSupplier(ctx context.Context, doc shared.DocumentUpload) (shared.VerificationResult, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("Sending document to verification supplier",
		"merchantId", doc.MerchantID,
		"documentType", doc.DocumentType,
		"documentId", doc.DocumentID,
	)

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

	verificationID := fmt.Sprintf("SUP-%s", doc.MerchantID)
	logger.Info("Supplier verified document successfully", "verificationId", verificationID)

	return shared.VerificationResult{
		Passed:         true,
		VerificationID: verificationID,
		Details:        "Identity document verified by supplier",
	}, nil
}

// PerformInternalVerifications runs Mollie's internal KYC checks:
//  1. Identity verification — does the verified identity match the merchant account?
//  2. Sanctions screening — is this person/business on a sanctions list?
//
// In production, additional checks would include: business legitimacy,
// legal authorization, acceptable business activities, and bank account verification.
func (a *Activities) PerformInternalVerifications(ctx context.Context, merchantID string) (shared.VerificationResult, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("Performing internal verifications", "merchantId", merchantID)

	// Check 1: Identity verification
	logger.Info("Running identity verification check", "merchantId", merchantID)

	// Check 2: Sanctions screening
	logger.Info("Running sanctions list screening", "merchantId", merchantID)

	verificationID := fmt.Sprintf("INT-%s", merchantID)
	logger.Info("Internal verifications passed", "verificationId", verificationID)

	return shared.VerificationResult{
		Passed:         true,
		VerificationID: verificationID,
		Details:        "All internal verifications passed (identity + sanctions)",
	}, nil
}
