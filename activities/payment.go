package activities

import (
	"context"
	"fmt"

	"go.temporal.io/sdk/activity"
)

// DisablePayments disables payment processing for a merchant
// who has not completed onboarding by the compliance deadline.
// Idempotency: naturally idempotent — setting status to "disabled" twice has the same effect.
func (a *Activities) DisablePayments(ctx context.Context, merchantID string) error {
	logger := activity.GetLogger(ctx)
	logger.Info("Disabling payments for merchant", "merchantId", merchantID)

	// In production: call payment services API to disable processing.
	logger.Info(fmt.Sprintf("Payments disabled successfully for merchant %s", merchantID))

	return nil
}
