package activities

// Activities is the receiver for all activity methods. Using a struct allows
// Temporal to auto-discover and register all methods via RegisterActivity(a),
// and lets us inject dependencies (e.g., API clients, DB connections) that
// each activity method can access through the receiver. In tests, stub fields
// can be toggled to avoid real side effects like sending emails or calling
// third-party APIs.
type Activities struct{}
