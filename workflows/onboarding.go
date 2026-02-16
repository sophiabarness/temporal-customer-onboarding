package workflows

import (
	"fmt"
	"time"

	"go.temporal.io/sdk/log"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"

	"temporal-customer-onboarding/shared"
)

// onboardingWorkflow holds workflow state and provides methods for each phase
// of the onboarding process.
type onboardingWorkflow struct {
	// Business state
	status     shared.OnboardingStatus
	documentID string
	startTime  time.Time

	// Workflow context
	req      shared.OnboardingRequest
	logger   log.Logger
	actCtx   workflow.Context
	signalCh workflow.ReceiveChannel
}

// newOnboardingWorkflow initializes the workflow struct, registers the query
// handler, and sets up the signal channel and activity options.
func newOnboardingWorkflow(ctx workflow.Context, req shared.OnboardingRequest) (*onboardingWorkflow, error) {
	w := &onboardingWorkflow{
		status:    shared.StatusPending,
		startTime: workflow.Now(ctx),
		req:       req,
		logger:    workflow.GetLogger(ctx),
		signalCh:  workflow.GetSignalChannel(ctx, shared.SignalDocumentSubmitted),
	}

	// Register query handler so external clients can check status.
	err := workflow.SetQueryHandler(ctx, shared.QueryOnboardingStatus, func() (shared.OnboardingStatusResponse, error) {
		elapsed := workflow.Now(ctx).Sub(w.startTime)
		daysRemaining := int((shared.DeadlineDay90 - elapsed).Hours() / 24)
		if daysRemaining < 0 {
			daysRemaining = 0
		}
		return shared.OnboardingStatusResponse{
			Status:        w.status,
			DaysRemaining: daysRemaining,
		}, nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to set query handler: %w", err)
	}

	// Configure activity options.
	actOpts := workflow.ActivityOptions{
		TaskQueue:           shared.ActivityTaskQueue,
		StartToCloseTimeout: 10 * time.Second,
		RetryPolicy: &temporal.RetryPolicy{
			MaximumAttempts: 3,
		},
	}
	w.actCtx = workflow.WithActivityOptions(ctx, actOpts)

	return w, nil
}

// waitForDocumentWithReminders sends periodic reminders and listens for the
// merchant's document submission signal. If the signal arrives during any
// reminder wait, reminders are cancelled and we proceed immediately.
func (w *onboardingWorkflow) waitForDocumentWithReminders(ctx workflow.Context) {
	w.status = shared.StatusRemindersActive

	reminders := []struct {
		delay        time.Duration
		reminderType string
	}{
		{shared.ReminderDay30, "day30"},
		{shared.ReminderDay60, "day60"},
	}

	for _, r := range reminders {
		timerCtx, timerCancel := workflow.WithCancel(ctx)
		timerFuture := workflow.NewTimer(timerCtx, r.delay)

		selector := workflow.NewSelector(ctx)

		selector.AddFuture(timerFuture, func(f workflow.Future) {
			_ = f.Get(ctx, nil)
		})

		selector.AddReceive(w.signalCh, func(ch workflow.ReceiveChannel, more bool) {
			ch.Receive(ctx, &w.documentID)
			w.logger.Info("Onboarding completion signal received during reminder phase",
				"merchantId", w.req.Merchant.MerchantID,
				"documentId", w.documentID,
			)
			timerCancel()
		})

		selector.Select(ctx)

		if w.documentID != "" {
			break
		}

		// Timer fired normally — send the reminder.
		reminderReq := shared.ReminderRequest{
			MerchantID:   w.req.Merchant.MerchantID,
			Email:        w.req.Merchant.Email,
			ReminderType: r.reminderType,
		}
		var reminderID string
		err := workflow.ExecuteActivity(w.actCtx, a.SendReminder, reminderReq).Get(ctx, &reminderID)
		if err != nil {
			w.logger.Error("Failed to send reminder", "error", err)
			// Continue — a failed reminder shouldn't block the onboarding process.
		}
		w.logger.Info("Reminder sent", "reminderType", r.reminderType, "reminderID", reminderID)
	}
}

// waitForDeadline waits for the remaining time until the 90-day deadline
// for the merchant to submit their document. If the signal arrives before
// the deadline, the workflow proceeds. Otherwise, the document remains empty.
func (w *onboardingWorkflow) waitForDeadline(ctx workflow.Context) {
	if w.documentID != "" {
		return // Already received during reminder phase.
	}

	remainingTime := shared.DeadlineDay90 - shared.ReminderDay60

	w.logger.Info("Waiting for onboarding completion before deadline",
		"merchantId", w.req.Merchant.MerchantID,
		"remainingTime", remainingTime,
	)

	workflow.AwaitWithTimeout(ctx, remainingTime, func() bool {
		return w.documentID != ""
	})

	// Drain any pending signal.
	for w.signalCh.ReceiveAsync(&w.documentID) {
	}
}

// disablePayments handles the case where the merchant missed the 90-day deadline.
// It disables payment processing and returns a failure result.
func (w *onboardingWorkflow) disablePayments(ctx workflow.Context) (string, error) {
	w.logger.Info("Onboarding deadline expired, disabling payments",
		"merchantId", w.req.Merchant.MerchantID,
	)
	w.status = shared.StatusPaymentsDisabled

	err := workflow.ExecuteActivity(w.actCtx, a.DisablePayments, w.req.Merchant.MerchantID).Get(ctx, nil)
	if err != nil {
		return "", fmt.Errorf("failed to disable payments: %w", err)
	}

	return fmt.Sprintf("ONBOARD-%s-PAYMENTS-DISABLED", w.req.Merchant.MerchantID), nil
}

// runKYC launches the identity verification child workflow and handles
// the result (approved or rejected).
func (w *onboardingWorkflow) runKYC(ctx workflow.Context) (string, error) {
	w.logger.Info("Merchant completed onboarding, starting KYC verification",
		"merchantId", w.req.Merchant.MerchantID,
	)
	w.status = shared.StatusKYCInProgress

	childOpts := workflow.ChildWorkflowOptions{
		WorkflowID: fmt.Sprintf("kyc-verify-%s", w.req.Merchant.MerchantID),
		TaskQueue:  shared.OnboardingWorkflowTaskQueue,
	}
	childCtx := workflow.WithChildOptions(ctx, childOpts)

	var kycResult shared.VerificationResult
	err := workflow.ExecuteChildWorkflow(childCtx, IdentityVerificationWorkflow, w.req.Merchant.MerchantID, w.documentID).Get(ctx, &kycResult)
	if err != nil {
		return "", fmt.Errorf("KYC child workflow failed: %w", err)
	}

	if !kycResult.Passed {
		w.status = shared.StatusRejected
		w.logger.Info("KYC verification failed",
			"merchantId", w.req.Merchant.MerchantID,
			"details", kycResult.Details,
		)

		// Notify merchant of rejection.
		reminderReq := shared.ReminderRequest{
			MerchantID:   w.req.Merchant.MerchantID,
			Email:        w.req.Merchant.Email,
			ReminderType: "kycRejection",
		}
		_ = workflow.ExecuteActivity(w.actCtx, a.SendReminder, reminderReq).Get(ctx, nil)

		return fmt.Sprintf("ONBOARD-%s-KYC-REJECTED", w.req.Merchant.MerchantID), nil
	}

	// Success — all checks passed.
	w.status = shared.StatusApproved
	w.logger.Info("Onboarding completed successfully",
		"merchantId", w.req.Merchant.MerchantID,
		"kycVerificationId", kycResult.VerificationID,
	)

	// Notify merchant of approval.
	reminderReq := shared.ReminderRequest{
		MerchantID:   w.req.Merchant.MerchantID,
		Email:        w.req.Merchant.Email,
		ReminderType: "onboardingApproved",
	}
	_ = workflow.ExecuteActivity(w.actCtx, a.SendReminder, reminderReq).Get(ctx, nil)

	return fmt.Sprintf("ONBOARD-%s-APPROVED", w.req.Merchant.MerchantID), nil
}

// OnboardingWorkflow models Mollie's merchant onboarding compliance process.
//
// Triggered when a merchant's first payment is received. The merchant can
// process payments immediately, but must complete onboarding within 90 days.
//
// Timeline:
//
//	Day 0  → Workflow starts (first payment received)
//	Day 30 → Send reminder
//	Day 60 → Send reminder
//	Day 90 → Deadline: if not completed, disable payments
//
// Temporal features demonstrated:
//   - Durable timers (workflow.Sleep for reminder schedule)
//   - Signals (SignalDocumentSubmitted)
//   - Queries (GetOnboardingStatus)
//   - Child workflows (Identity Verification)
//   - Retry policies with non-retryable error types
func OnboardingWorkflow(ctx workflow.Context, req shared.OnboardingRequest) (string, error) {
	w, err := newOnboardingWorkflow(ctx, req)
	if err != nil {
		return "", err
	}

	w.logger.Info("Onboarding workflow started",
		"merchantId", req.Merchant.MerchantID,
	)

	// Phase 1: Send reminders while waiting for document submission.
	w.waitForDocumentWithReminders(ctx)

	// Phase 2: Wait for the 90-day deadline if document not yet received.
	w.waitForDeadline(ctx)

	// Phase 3: Outcome — either disable payments or run KYC.
	if w.documentID == "" {
		return w.disablePayments(ctx)
	}
	return w.runKYC(ctx)
}
