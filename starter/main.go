package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"go.temporal.io/sdk/client"

	"temporal-customer-onboarding/shared"
	"temporal-customer-onboarding/workflows"
)

func main() {
	c, err := client.Dial(client.Options{})
	if err != nil {
		log.Fatalf("Unable to create Temporal client: %v", err)
	}
	defer c.Close()

	merchantID := "MERCH-001"
	// Business-meaningful workflow ID acts as an idempotency key: prevents
	// duplicate onboarding for the same merchant while a workflow is running.
	workflowID := fmt.Sprintf("onboard-merchant-%s", merchantID)
	reader := bufio.NewReader(os.Stdin)

	req := shared.OnboardingRequest{
		Merchant: shared.MerchantInfo{
			MerchantID:   merchantID,
			Name:         "Acme Online Store",
			Email:        "onboarding@acme-store.com",
			Country:      "NL",
			BusinessType: "ecommerce",
		},
	}

	fmt.Println()
	fmt.Println("🚀 Starting onboarding workflow for merchant", merchantID)

	// To also reject re-onboarding after completion, add WorkflowIDReusePolicy:
	//   enums.WORKFLOW_ID_REUSE_POLICY_REJECT_DUPLICATE
	we, err := c.ExecuteWorkflow(
		context.Background(),
		client.StartWorkflowOptions{
			ID:        workflowID,
			TaskQueue: shared.OnboardingWorkflowTaskQueue,
		},
		workflows.OnboardingWorkflow,
		req,
	)
	if err != nil {
		log.Fatalf("Unable to start workflow: %v", err)
	}
	fmt.Printf("   WorkflowID: %s\n", we.GetID())
	fmt.Printf("   RunID:      %s\n", we.GetRunID())

	// Interactive menu loop.
	for {
		fmt.Println()
		fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
		fmt.Println("  Merchant Onboarding CLI")
		fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
		fmt.Println()
		fmt.Println("  [1] Submit document (triggers KYC verification)")
		fmt.Println("  [2] Query workflow status")
		fmt.Println("  [3] Exit (workflow continues running)")
		fmt.Println()
		fmt.Print("Choose: ")

		choice, _ := reader.ReadString('\n')
		choice = strings.TrimSpace(choice)

		switch choice {
		case "1":
			handleSubmitDocument(c, workflowID, we, reader)
			return

		case "2":
			handleQueryStatus(c, workflowID)

		case "3":
			fmt.Println()
			fmt.Println("👋 Exiting CLI. The workflow continues running in Temporal.")
			fmt.Println("   Re-run this program to reconnect, or view at http://localhost:8233")
			return

		default:
			fmt.Println("❌ Invalid choice. Please enter 1, 2, or 3.")
		}
	}
}

func handleSubmitDocument(c client.Client, workflowID string, we client.WorkflowRun, reader *bufio.Reader) {
	fmt.Println()
	fmt.Print("Enter your government ID document number (digits only to pass): ")
	documentID, _ := reader.ReadString('\n')
	documentID = strings.TrimSpace(documentID)

	fmt.Printf("\n📤 Sending document ID '%s' to workflow...\n", documentID)

	err := c.SignalWorkflow(
		context.Background(),
		workflowID,
		"",
		shared.SignalDocumentSubmitted,
		documentID,
	)
	if err != nil {
		log.Fatalf("Unable to signal workflow: %v", err)
	}
	fmt.Println("✅ Signal sent! Waiting for workflow result...")
	fmt.Println()

	// Wait for the workflow to complete.
	var result string
	err = we.Get(context.Background(), &result)
	if err != nil {
		log.Fatalf("Workflow failed: %v", err)
	}

	fmt.Printf("🏁 Result: %s\n", result)
}

func handleQueryStatus(c client.Client, workflowID string) {
	resp, err := c.QueryWorkflow(
		context.Background(),
		workflowID,
		"",
		shared.QueryOnboardingStatus,
	)
	if err != nil {
		fmt.Printf("❌ Query failed: %v\n", err)
		return
	}

	var statusResp shared.OnboardingStatusResponse
	if err := resp.Get(&statusResp); err != nil {
		fmt.Printf("❌ Failed to decode status: %v\n", err)
		return
	}

	fmt.Printf("\n📋 Status: %s (%d days remaining)\n", statusResp.Status, statusResp.DaysRemaining)
}
