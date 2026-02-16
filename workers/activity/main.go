package main

import (
	"log"

	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"

	"temporal-customer-onboarding/activities"
	"temporal-customer-onboarding/shared"
)

func main() {
	// Connect to the Temporal server via gRPC.
	// Key client.Options for production:
	//   HostPort (default: localhost:7233) — address of the Temporal server or Temporal Cloud endpoint.
	//   Namespace (default: "default") — logical isolation for workflows, like a database schema.
	//     Different teams/environments get their own namespace (e.g., "onboarding-prod", "onboarding-staging").
	//   ConnectionOptions.TLS — mTLS config required for Temporal Cloud or secure self-hosted clusters.
	//   MetricsHandler — export metrics to Prometheus/Datadog for schedule-to-start latencies, task slots, etc.
	c, err := client.Dial(client.Options{})
	if err != nil {
		log.Fatalf("Unable to create Temporal client: %v", err)
	}
	defer c.Close()

	// Key worker.Options for production tuning:
	//   MaxConcurrentActivityExecutionSize (default: 1000) — max activities running in parallel on this worker.
	//     Tune down to protect rate-limited downstream services (e.g., KYC supplier API).
	//   TaskQueueActivitiesPerSecond (default: 100k) — server-enforced rate limit across ALL workers on this queue.
	//     Use this for global rate limiting regardless of how many workers are running.
	//   WorkerStopTimeout (default: 0s) — graceful shutdown window for in-flight activities during deploys.
	//   MaxConcurrentActivityTaskPollers (default: 2) — goroutines polling the server for new tasks.
	w := worker.New(c, shared.ActivityTaskQueue, worker.Options{})

	// Register all activity methods via the struct.
	// In production, inject real dependencies here (e.g., API keys, DB connections).
	a := &activities.Activities{}
	w.RegisterActivity(a)

	log.Println("Starting activity worker...")
	if err := w.Run(worker.InterruptCh()); err != nil {
		log.Fatalf("Unable to start worker: %v", err)
	}
}
