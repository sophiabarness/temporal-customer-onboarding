package activities

import (
	"context"
	"fmt"

	"go.temporal.io/sdk/activity"

	"temporal-customer-onboarding/shared"
)

// SendReminder sends an onboarding reminder email to the merchant.
// Idempotency: not naturally idempotent (retries would send duplicate emails).
// In production, pass reminderID as an idempotency key to the email provider.
func (a *Activities) SendReminder(ctx context.Context, req shared.ReminderRequest) (string, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("Sending reminder",
		"merchantId", req.MerchantID,
		"reminderType", req.ReminderType,
		"email", req.Email,
	)

	// In production: integrate with email service (SendGrid, SES, etc.)
	reminderID := fmt.Sprintf("REMIND-%s-%s", req.MerchantID, req.ReminderType)
	logger.Info("Reminder sent successfully", "reminderID", reminderID)

	return reminderID, nil
}
