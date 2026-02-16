package tests

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/testsuite"

	"temporal-customer-onboarding/activities"
	"temporal-customer-onboarding/shared"
)

func TestSendReminder(t *testing.T) {
	testSuite := &testsuite.WorkflowTestSuite{}
	env := testSuite.NewTestActivityEnvironment()
	a := &activities.Activities{}
	env.RegisterActivity(a.SendReminder)

	req := shared.ReminderRequest{
		MerchantID:   "MERCH-001",
		Email:        "test@example.com",
		ReminderType: "day30",
	}

	result, err := env.ExecuteActivity(a.SendReminder, req)
	assert.NoError(t, err)

	var reminderID string
	err = result.Get(&reminderID)
	assert.NoError(t, err)
	assert.Equal(t, "REMIND-MERCH-001-day30", reminderID)
}

func TestDisablePayments(t *testing.T) {
	testSuite := &testsuite.WorkflowTestSuite{}
	env := testSuite.NewTestActivityEnvironment()
	a := &activities.Activities{}
	env.RegisterActivity(a.DisablePayments)

	_, err := env.ExecuteActivity(a.DisablePayments, "MERCH-001")
	assert.NoError(t, err)
}

func TestValidateWithSupplier_NumericID_Passes(t *testing.T) {
	testSuite := &testsuite.WorkflowTestSuite{}
	env := testSuite.NewTestActivityEnvironment()
	a := &activities.Activities{}
	env.RegisterActivity(a.ValidateWithSupplier)

	doc := shared.DocumentUpload{
		MerchantID:   "MERCH-001",
		DocumentType: "governmentId",
		DocumentID:   "123456789",
	}

	result, err := env.ExecuteActivity(a.ValidateWithSupplier, doc)
	assert.NoError(t, err)

	var verResult shared.VerificationResult
	err = result.Get(&verResult)
	assert.NoError(t, err)
	assert.True(t, verResult.Passed)
	assert.Equal(t, "SUP-MERCH-001", verResult.VerificationID)
}

func TestValidateWithSupplier_NonNumericID_Rejected(t *testing.T) {
	testSuite := &testsuite.WorkflowTestSuite{}
	env := testSuite.NewTestActivityEnvironment()
	a := &activities.Activities{}
	env.RegisterActivity(a.ValidateWithSupplier)

	doc := shared.DocumentUpload{
		MerchantID:   "MERCH-001",
		DocumentType: "governmentId",
		DocumentID:   "ABC123", // Contains letters — should be rejected.
	}

	_, err := env.ExecuteActivity(a.ValidateWithSupplier, doc)
	assert.Error(t, err)

	// Verify it's a non-retryable application error.
	var appErr *temporal.ApplicationError
	assert.ErrorAs(t, err, &appErr)
}

func TestPerformInternalVerifications(t *testing.T) {
	testSuite := &testsuite.WorkflowTestSuite{}
	env := testSuite.NewTestActivityEnvironment()
	a := &activities.Activities{}
	env.RegisterActivity(a.PerformInternalVerifications)

	result, err := env.ExecuteActivity(a.PerformInternalVerifications, "MERCH-001")
	assert.NoError(t, err)

	var verResult shared.VerificationResult
	err = result.Get(&verResult)
	assert.NoError(t, err)
	assert.True(t, verResult.Passed)
	assert.Equal(t, "INT-MERCH-001", verResult.VerificationID)
}
