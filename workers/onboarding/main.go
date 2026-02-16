package main

import (
	"log"

	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"

	"temporal-customer-onboarding/shared"
	"temporal-customer-onboarding/workflows"
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
	//   MaxConcurrentWorkflowTaskExecutionSize (default: 1000) — max workflow tasks running in parallel.
	//     Workflow tasks are lightweight (no I/O), so the default is usually fine.
	//   StickyScheduleToStartTimeout (default: 5s) — Temporal tries to route workflow tasks back to the
	//     same worker that has state cached in memory. If that worker is unavailable for longer than this
	//     timeout, the task is reassigned to another worker which replays from history.
	w := worker.New(c, shared.OnboardingWorkflowTaskQueue, worker.Options{})

	// Register workflows.
	w.RegisterWorkflow(workflows.OnboardingWorkflow)
	w.RegisterWorkflow(workflows.IdentityVerificationWorkflow)

	log.Println("Starting onboarding workflow worker...")
	if err := w.Run(worker.InterruptCh()); err != nil {
		log.Fatalf("Unable to start worker: %v", err)
	}
}
