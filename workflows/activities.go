package workflows

import "temporal-customer-onboarding/activities"

// a is the activities struct used by workflows to reference activity methods.
// The actual struct is registered with the worker; this variable is only used
// to provide method references for workflow.ExecuteActivity calls.
var a *activities.Activities
