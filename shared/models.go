package shared

// OnboardingStatus represents the current state of merchant onboarding.
type OnboardingStatus string

const (
	StatusPending          OnboardingStatus = "PENDING"
	StatusRemindersActive  OnboardingStatus = "AWAITING_KYC_DOCUMENTS"
	StatusKYCInProgress    OnboardingStatus = "KYC_IN_PROGRESS"
	StatusApproved         OnboardingStatus = "APPROVED"
	StatusRejected         OnboardingStatus = "REJECTED"
	StatusPaymentsDisabled OnboardingStatus = "PAYMENTS_DISABLED"
)

// OnboardingStatusResponse is returned by the query handler.
type OnboardingStatusResponse struct {
	Status        OnboardingStatus `json:"status"`
	DaysRemaining int              `json:"daysRemaining"`
}

// MerchantInfo contains the merchant's registration details.
type MerchantInfo struct {
	MerchantID   string `json:"merchantId"`
	Name         string `json:"name"`
	Email        string `json:"email"`
	Country      string `json:"country"`
	BusinessType string `json:"businessType"`
}

// OnboardingRequest is the input to the OnboardingWorkflow.
type OnboardingRequest struct {
	Merchant MerchantInfo `json:"merchant"`
}

// ReminderRequest is the input to the SendReminder activity.
type ReminderRequest struct {
	MerchantID   string `json:"merchantId"`
	Email        string `json:"email"`
	ReminderType string `json:"reminderType"` // "day30", "day60", "final"
}

// DocumentUpload represents a document submitted by the merchant.
type DocumentUpload struct {
	MerchantID   string `json:"merchantId"`
	DocumentType string `json:"documentType"` // "governmentId", "proofOfAddress"
	DocumentID   string `json:"documentId"`
}

// VerificationResult is the output from verification activities.
type VerificationResult struct {
	Passed         bool   `json:"passed"`
	VerificationID string `json:"verificationId"`
	Details        string `json:"details"`
}
