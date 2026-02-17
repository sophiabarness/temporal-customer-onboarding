package workflows

import (
	"fmt"
	"time"

	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"

	"temporal-customer-onboarding/shared"
)

// IdentityVerificationWorkflow is a child workflow that orchestrates KYC verification.
// It validates a merchant's identity document with a 3rd party supplier,
// then runs internal identity verification.
func IdentityVerificationWorkflow(ctx workflow.Context, merchantID string, documentID string) (shared.VerificationResult, error) {
	logger := workflow.GetLogger(ctx)
	logger.Info("Identity verification workflow started",
		"merchantId", merchantID,
		"documentId", documentID,
	)

	// Activity options for ValidateWithSupplier.
	// External API calls get more time and retries.
	supplierOpts := workflow.ActivityOptions{
		TaskQueue:           shared.ActivityTaskQueue,
		StartToCloseTimeout: 30 * time.Second,
		RetryPolicy: &temporal.RetryPolicy{
			InitialInterval:        time.Second,
			BackoffCoefficient:     2.0,
			MaximumInterval:        30 * time.Second,
			NonRetryableErrorTypes: []string{shared.ErrTypeIdentityVerificationFailed},
		},
	}

	// Activity options for PerformInternalVerifications.
	// Internal database checks — faster and more reliable than external APIs.
	internalOpts := workflow.ActivityOptions{
		TaskQueue:           shared.ActivityTaskQueue,
		StartToCloseTimeout: 10 * time.Second,
		RetryPolicy: &temporal.RetryPolicy{
			MaximumAttempts: 2,
		},
	}

	// Step 1: Validate with 3rd party supplier.
	doc := shared.DocumentUpload{
		MerchantID:   merchantID,
		DocumentType: "governmentId",
		DocumentID:   documentID, // User-provided document ID from the signal.
	}

	var supplierResult shared.VerificationResult
	supplierCtx := workflow.WithActivityOptions(ctx, supplierOpts)
	err := workflow.ExecuteActivity(supplierCtx, a.ValidateWithSupplier, doc).Get(ctx, &supplierResult)
	if err != nil {
		logger.Error("Supplier validation failed", "merchantId", merchantID, "error", err)
		return shared.VerificationResult{
			Passed:         false,
			VerificationID: fmt.Sprintf("KYC-FAIL-%s", merchantID),
			Details:        fmt.Sprintf("Supplier validation failed: %v", err),
		}, nil // Return result, not error — KYC rejection is a business outcome, not a workflow failure.
	}
	logger.Info("Supplier validation passed", "verificationId", supplierResult.VerificationID)

	// Step 2: Perform internal identity verification.
	var internalResult shared.VerificationResult
	internalCtx := workflow.WithActivityOptions(ctx, internalOpts)
	err = workflow.ExecuteActivity(internalCtx, a.PerformInternalVerifications, merchantID).Get(ctx, &internalResult)
	if err != nil {
		logger.Error("Internal verifications failed", "merchantId", merchantID, "error", err)
		return shared.VerificationResult{
			Passed:         false,
			VerificationID: fmt.Sprintf("KYC-FAIL-%s", merchantID),
			Details:        fmt.Sprintf("Internal verifications failed: %v", err),
		}, nil
	}
	logger.Info("Internal verifications passed", "verificationId", internalResult.VerificationID)

	// All checks passed.
	return shared.VerificationResult{
		Passed:         true,
		VerificationID: fmt.Sprintf("KYC-%s", merchantID),
		Details:        "All KYC checks passed (supplier + internal)",
	}, nil
}
