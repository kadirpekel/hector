package cli

import (
	"context"
	"fmt"
)

// TaskGetCommand retrieves task details
func TaskGetCommand(args Args) error {
	// Create client
	a2aClient, err := createClient(args)
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}
	defer a2aClient.Close()

	// Get task
	task, err := a2aClient.GetTask(context.Background(), args.AgentID, args.TaskID)
	if err != nil {
		return fmt.Errorf("failed to get task: %w", err)
	}

	// Display
	DisplayTask(task)

	return nil
}

// TaskCancelCommand cancels a running task
func TaskCancelCommand(args Args) error {
	// Create client
	a2aClient, err := createClient(args)
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}
	defer a2aClient.Close()

	// Cancel task
	task, err := a2aClient.CancelTask(context.Background(), args.AgentID, args.TaskID)
	if err != nil {
		return fmt.Errorf("failed to cancel task: %w", err)
	}

	fmt.Printf("✅ Task cancelled successfully\n\n")

	// Display updated task
	DisplayTask(task)

	return nil
}
