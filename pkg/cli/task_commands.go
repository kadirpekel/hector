package cli

import (
	"context"
	"fmt"
)

// TaskGetCommand retrieves task details
func TaskGetCommand(args *CLIArgs) error {
	a2aClient, err := createRuntimeClient(args)
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}
	defer a2aClient.Close()

	task, err := a2aClient.GetTask(context.Background(), args.AgentID, args.TaskID)
	if err != nil {
		return fmt.Errorf("failed to get task: %w", err)
	}

	DisplayTask(task)
	return nil
}

// TaskCancelCommand cancels a running task
func TaskCancelCommand(args *CLIArgs) error {
	a2aClient, err := createRuntimeClient(args)
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}
	defer a2aClient.Close()

	task, err := a2aClient.CancelTask(context.Background(), args.AgentID, args.TaskID)
	if err != nil {
		return fmt.Errorf("failed to cancel task: %w", err)
	}

	fmt.Printf("✅ Task cancelled successfully\n\n")
	DisplayTask(task)
	return nil
}
