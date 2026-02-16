package tests

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.temporal.io/sdk/testsuite"

	"temporal-customer-onboarding/activities"
	"temporal-customer-onboarding/shared"
	"temporal-customer-onboarding/workflows"
)

func defaultOnboardingRequest() shared.OnboardingRequest {
	return shared.OnboardingRequest{
		Merchant: shared.MerchantInfo{
			MerchantID:   "MERCH-001",
			Name:         "Test Store",
			Email:        "test@example.com",
			Country:      "NL",
			BusinessType: "ecommerce",
		},
		FirstPaymentDate: time.Now(),
	}
}

func registerMockActivities(env *testsuite.TestWorkflowEnvironment) *activities.Activities {
	a := &activities.Activities{}
	env.RegisterActivity(a)
	return a
}

func TestOnboardingWorkflow_HappyPath(t *testing.T) {
	testSuite := &testsuite.WorkflowTestSuite{}
	env := testSuite.NewTestWorkflowEnvironment()
	a := registerMockActivities(env)

	env.OnActivity(a.SendReminder, mock.Anything, mock.Anything).Return("REMIND-001", nil)
	env.OnActivity(a.DisablePayments, mock.Anything, mock.Anything).Return(nil)

	// Mock child workflow — KYC passes.
	env.OnWorkflow(workflows.IdentityVerificationWorkflow, mock.Anything, mock.Anything, mock.Anything).Return(
		shared.VerificationResult{
			Passed:         true,
			VerificationID: "KYC-MERCH-001",
			Details:        "All checks passed",
		}, nil,
	)

	// Send completion signal with document ID after a short delay.
	env.RegisterDelayedCallback(func() {
		env.SignalWorkflow(shared.SignalDocumentSubmitted, "123456789")
	}, time.Millisecond*100)

	req := defaultOnboardingRequest()
	env.ExecuteWorkflow(workflows.OnboardingWorkflow, req)

	assert.True(t, env.IsWorkflowCompleted())
	assert.NoError(t, env.GetWorkflowError())

	var result string
	assert.NoError(t, env.GetWorkflowResult(&result))
	assert.Equal(t, "ONBOARD-MERCH-001-APPROVED", result)
}

func TestOnboardingWorkflow_Timeout_DisablesPayments(t *testing.T) {
	testSuite := &testsuite.WorkflowTestSuite{}
	env := testSuite.NewTestWorkflowEnvironment()
	a := registerMockActivities(env)

	env.OnActivity(a.SendReminder, mock.Anything, mock.Anything).Return("REMIND-001", nil)
	env.OnActivity(a.DisablePayments, mock.Anything, mock.Anything).Return(nil)

	// No signal sent — merchant never completes onboarding.

	req := defaultOnboardingRequest()
	env.ExecuteWorkflow(workflows.OnboardingWorkflow, req)

	assert.True(t, env.IsWorkflowCompleted())
	assert.NoError(t, env.GetWorkflowError())

	var result string
	assert.NoError(t, env.GetWorkflowResult(&result))
	assert.Equal(t, "ONBOARD-MERCH-001-PAYMENTS-DISABLED", result)
}

func TestOnboardingWorkflow_KYCRejection(t *testing.T) {
	testSuite := &testsuite.WorkflowTestSuite{}
	env := testSuite.NewTestWorkflowEnvironment()
	a := registerMockActivities(env)

	env.OnActivity(a.SendReminder, mock.Anything, mock.Anything).Return("REMIND-001", nil)

	// Mock child workflow — KYC fails.
	env.OnWorkflow(workflows.IdentityVerificationWorkflow, mock.Anything, mock.Anything, mock.Anything).Return(
		shared.VerificationResult{
			Passed:         false,
			VerificationID: "KYC-FAIL-MERCH-001",
			Details:        "Supplier rejected identity document",
		}, nil,
	)

	// Send completion signal with non-numeric document ID.
	env.RegisterDelayedCallback(func() {
		env.SignalWorkflow(shared.SignalDocumentSubmitted, "ABC123")
	}, time.Millisecond*100)

	req := defaultOnboardingRequest()
	env.ExecuteWorkflow(workflows.OnboardingWorkflow, req)

	assert.True(t, env.IsWorkflowCompleted())
	assert.NoError(t, env.GetWorkflowError())

	var result string
	assert.NoError(t, env.GetWorkflowResult(&result))
	assert.Equal(t, "ONBOARD-MERCH-001-KYC-REJECTED", result)
}

func TestOnboardingWorkflow_QueryStatus(t *testing.T) {
	testSuite := &testsuite.WorkflowTestSuite{}
	env := testSuite.NewTestWorkflowEnvironment()
	a := registerMockActivities(env)

	env.OnActivity(a.SendReminder, mock.Anything, mock.Anything).Return("REMIND-001", nil)
	env.OnActivity(a.DisablePayments, mock.Anything, mock.Anything).Return(nil)

	// Query status before the workflow completes.
	env.RegisterDelayedCallback(func() {
		result, err := env.QueryWorkflow(shared.QueryOnboardingStatus)
		assert.NoError(t, err)
		var statusResp shared.OnboardingStatusResponse
		assert.NoError(t, result.Get(&statusResp))
		assert.True(t,
			statusResp.Status == shared.StatusPending || statusResp.Status == shared.StatusRemindersActive,
			"Expected PENDING or AWAITING_KYC_DOCUMENTS, got %s", statusResp.Status,
		)
		assert.True(t, statusResp.DaysRemaining >= 0, "Days remaining should be non-negative")
	}, time.Millisecond*50)

	req := defaultOnboardingRequest()
	env.ExecuteWorkflow(workflows.OnboardingWorkflow, req)

	assert.True(t, env.IsWorkflowCompleted())
}
