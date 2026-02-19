package tests

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.temporal.io/sdk/testsuite"

	"temporal-customer-onboarding/shared"
	"temporal-customer-onboarding/workflows"
)

func TestOnboardingWorkflow_SignalDuringDeadlinePhase(t *testing.T) {
	testSuite := &testsuite.WorkflowTestSuite{}
	env := testSuite.NewTestWorkflowEnvironment()
	a := registerMockActivities(env)

	// Mock reminder activity (should be called twice: Day 30 and Day 60)
	env.OnActivity(a.SendReminder, mock.Anything, mock.Anything).Return("REMIND-001", nil)

	// Mock child workflow (KYC) - we expect this to run if the signal is processed!
	env.OnWorkflow(workflows.IdentityVerificationWorkflow, mock.Anything, mock.Anything, mock.Anything).Return(
		shared.VerificationResult{
			Passed:         true,
			VerificationID: "KYC-MERCH-001",
			Details:        "All checks passed",
		}, nil,
	)

	// Register a delayed signal that arrives at Day 70
	// (After the 60 days of reminders, but before the 90 day deadline)
	env.RegisterDelayedCallback(func() {
		env.SignalWorkflow(shared.SignalDocumentSubmitted, "123456789")
	}, time.Hour*24*70) // Day 70

	startTime := env.Now()
	req := defaultOnboardingRequest()
	env.ExecuteWorkflow(workflows.OnboardingWorkflow, req)

	assert.True(t, env.IsWorkflowCompleted())
	assert.NoError(t, env.GetWorkflowError())

	// We expect the workflow to SUCCEED because we sent the document before Day 90.
	var result string
	env.GetWorkflowResult(&result)
	assert.Equal(t, "ONBOARD-MERCH-001-APPROVED", result, "Workflow should have approved the user, but got: %s", result)

	// Additional Check: Verify TIMING
	// The signal was sent at Day 70.
	// If the workflow processes it immediately, total duration should be ~70 days.
	// If the workflow ignores it until deadline, total duration will be ~90 days.
	duration := env.Now().Sub(startTime)
	t.Logf("Workflow duration: %v hours", duration.Hours())

	// 70 days = 1680 hours.
	// 90 days = 2160 hours.
	// If duration > 2000 hours (approx 83 days), we have a bug!
	assert.Less(t, duration.Hours(), 2000.0, "Workflow took too long! It likely ignored the signal until deadline.")
}
