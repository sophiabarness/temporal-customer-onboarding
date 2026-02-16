package shared

import "time"

// Task queue names.
const (
	OnboardingWorkflowTaskQueue = "onboarding-workflow-tq"
	ActivityTaskQueue           = "activity-tq"
)

// Signal and query names.
const (
	SignalDocumentSubmitted = "signal-document-submitted"
	SignalDocumentUploaded  = "signal-document-uploaded"
	QueryOnboardingStatus   = "query-onboarding-status"
)

// Compliance timeline constants.
const (
	ReminderDay30 = 30 * 24 * time.Hour
	ReminderDay60 = 60 * 24 * time.Hour
	DeadlineDay90 = 90 * 24 * time.Hour
)

// Error types for non-retryable failures.
const (
	ErrTypeIdentityVerificationFailed = "IdentityVerificationFailed"
)
