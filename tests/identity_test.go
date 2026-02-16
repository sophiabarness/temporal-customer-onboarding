package tests

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.temporal.io/sdk/testsuite"

	"temporal-customer-onboarding/activities"
	"temporal-customer-onboarding/shared"
	"temporal-customer-onboarding/workflows"
)

func TestIdentityVerificationWorkflow_HappyPath(t *testing.T) {
	testSuite := &testsuite.WorkflowTestSuite{}
	env := testSuite.NewTestWorkflowEnvironment()
	a := &activities.Activities{}

	env.RegisterActivity(a.ValidateWithSupplier)
	env.RegisterActivity(a.PerformInternalVerifications)

	env.OnActivity(a.ValidateWithSupplier, mock.Anything, mock.Anything).Return(
		shared.VerificationResult{
			Passed:         true,
			VerificationID: "SUP-MERCH-001",
			Details:        "Supplier verified",
		}, nil,
	)

	env.OnActivity(a.PerformInternalVerifications, mock.Anything, mock.Anything).Return(
		shared.VerificationResult{
			Passed:         true,
			VerificationID: "INT-MERCH-001",
			Details:        "Internal checks passed",
		}, nil,
	)

	env.ExecuteWorkflow(workflows.IdentityVerificationWorkflow, "MERCH-001", "123456789")

	assert.True(t, env.IsWorkflowCompleted())
	assert.NoError(t, env.GetWorkflowError())

	var result shared.VerificationResult
	assert.NoError(t, env.GetWorkflowResult(&result))
	assert.True(t, result.Passed)
	assert.Equal(t, "KYC-MERCH-001", result.VerificationID)
}

func TestIdentityVerificationWorkflow_SupplierRejection(t *testing.T) {
	testSuite := &testsuite.WorkflowTestSuite{}
	env := testSuite.NewTestWorkflowEnvironment()
	a := &activities.Activities{}

	env.RegisterActivity(a.ValidateWithSupplier)
	env.RegisterActivity(a.PerformInternalVerifications)

	env.OnActivity(a.ValidateWithSupplier, mock.Anything, mock.Anything).Return(
		shared.VerificationResult{},
		assert.AnError,
	)

	env.ExecuteWorkflow(workflows.IdentityVerificationWorkflow, "MERCH-001", "ABC123")

	assert.True(t, env.IsWorkflowCompleted())
	assert.NoError(t, env.GetWorkflowError())

	var result shared.VerificationResult
	assert.NoError(t, env.GetWorkflowResult(&result))
	assert.False(t, result.Passed)
	assert.Contains(t, result.Details, "Supplier validation failed")
}
