# Customer Onboarding Workflow with Temporal

A Temporal application modeling a merchant onboarding compliance process. When a merchant's first payment arrives, they can process payments immediately but must complete Know-Your-Customer (KYC) verification within 90 days or payments are disabled.

> Inspired by [Mollie Payments' adoption of Temporal](https://youtu.be/5bc3MXuZTRc?si=F6FuWe8b_ioc97m6) for long-running onboarding workflows across European financial services.

## Overview

<img src="docs/onboarding-workflow.png" alt="Customer onboarding flow">
<img src="docs/overview.png" alt="Onboarding workflow diagram" width="600">



**Timeline:**
- **Day 0** — Workflow starts when the merchant's first payment is received
- **Day 30** — Reminder sent if document not yet submitted
- **Day 60** — Second reminder
- **Day 90** — Deadline: if no document submitted, payments are disabled

If the merchant submits their document at any point, the reminders stop and KYC verification begins via a child workflow.

## Without Temporal

1.  **State Machine via Polling**:
    *   **Document Upload API Endpoint**: Updates DB state and pushes a verification job to a queue.
    *   **Queue Consumer (Async KYC)**: Pulls the job and calls the 3rd-party identity provider. If the API is down, it retries N times before failing (Dead Letter Queue).
    *   **Cron A (Reminders)**: Polls every hour: `SELECT * FROM merchants WHERE created_at < NOW() - 30_DAYS AND reminder_sent = false`.
    *   **Cron B (Deadline)**: Polls every hour: `SELECT * FROM merchants WHERE created_at < NOW() - 90_DAYS AND payments_enabled = true`.

2.  **Pain Points**:
    *   **Race Conditions**: A user uploads a document just seconds before the deadline. The `Document Upload API` starts processing, but the `Deadline Cron` fires at the same moment. The cron disables the user, but the `Document Upload API` then overwrites the status to "Verifying". Result: The user is in a broken state (payments disabled, but UI says verifying).
    *   **Visibility Problems**: To debug why a merchant was disabled, you'd have to grep logs across 3 different systems (Document Upload, Reminder Cron, Deadline Cron).
    *   **Rigid Retries**: Standard queues often have hardcoded retry limits. 
    *   **Boilerplate**: You write 80% infrastructure code (crons, locks, queue consumers) and 20% business logic.


## How Temporal Solves Challenges

| Challenge | Temporal Feature | Impact |
|---|---|---|
| Long-running state across 90 days | **Durable Timers** | No cron jobs, no polling — timers survive crashes and restarts |
| Unreliable third-party KYC APIs | **Activity Retries & Timeouts** | Automatic retry with backoff; clean separation from business logic |
| Waiting for async user input | **Signals** (`SignalDocumentSubmitted`) | Workflow sleeps until event arrives — no polling, no message queues |
| Race conditions (signal vs. timer) | **Selectors** | Timers and signals race cleanly; no distributed locks needed |
| Querying workflow state | **Queries** (`QueryOnboardingStatus`) | Real-time status without an external database |
| Isolating failure domains | **Child Workflows** | KYC verification has its own retry policy and lifecycle |
| Business vs. infrastructure errors | **Non-Retryable Errors** | Supplier rejections fail fast; transient errors retry automatically |


## Design Decisions & Best Practices

- **Workflow ID as idempotency key** — `onboard-merchant-{id}` uses a meaningful business identifier to prevent duplicate onboarding for the same merchant.
- **Child workflow for KYC** — Isolates verification with its own retry policy and timeout. Can be reused for annual re-verification without duplicating logic.
- **Business outcomes as return values** — KYC rejection returns `VerificationResult{Passed: false}`, not a workflow error. `NonRetryableApplicationError` is used to distinguish business rejections from transient failures.
- **Signals for external events** — Signals deliver data into a running workflow without polling a database or queue.
- **Queries for state, not a database** — Workflow state is already durable. Expose it via query handlers instead of writing to an external store.
- **Selectors to race timers against signals** — Lets the workflow respond immediately to events instead of waiting for a timer to expire.
- **Separate workflow and activity workers** — Workflow worker is CPU-light (timers/signals); activity worker is I/O-bound (API calls). Scale independently in production.
- **Struct-based activities for dependency injection** — Register activities on a struct so dependencies can be injected at startup and swapped in tests.

## Getting Started

### Prerequisites
- Go 1.21+
- [Temporal CLI](https://docs.temporal.io/cli) (`brew install temporal`)

### Run

```bash
# Terminal 1: Temporal dev server
temporal server start-dev

# Terminal 2: Workflow worker
go run ./workers/onboarding/main.go

# Terminal 3: Activity worker
go run ./workers/activity/main.go

# Terminal 4: Interactive CLI
go run ./starter/main.go
```

**Demo paths:**
- Submit a **numeric** document → KYC passes → `APPROVED`
- Submit a **non-numeric** document → supplier rejects → `KYC-REJECTED`
- Don't submit → reminders fire at Day 30/60 → deadline expires → `PAYMENTS-DISABLED`
- **Fault Tolerance**:
    1.  Start the workflow: `go run ./starter`
    2.  Simulate a crash: Kill the `go run ./workers/activity` process during execution.
    3.  Submit a document. The workflow will wait for an activity worker without losing state.
    4.  Restart the worker. The workflow resumes immediately.
    5.  **Chaos Testing**: Submit any numeric document ID (e.g., `12345`). The activity has a built-in **75% failure rate** to simulate a generalized outage. Watch the Temporal Web UI to see automatic retries in action.

### Test
```bash
go test ./tests/ -v
```

### Observe
Open http://localhost:8233 to view workflow execution, event history, and query results in the Temporal Web UI.
