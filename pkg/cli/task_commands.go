package cli

import (
	"context"
	"fmt"

	"github.com/kadirpekel/hector/pkg/config"
)

// TaskGetCommand retrieves task details
func TaskGetCommand(args *CLIArgs, cfg *config.Config) error {
	client, err := createClient(args, cfg)
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}
	defer client.Close()

	task, err := client.GetTask(context.Background(), args.AgentID, args.TaskID)
	if err != nil {
		return fmt.Errorf("failed to get task: %w", err)
	}

	DisplayTask(task)
	return nil
}

// TaskCancelCommand cancels a running task
func TaskCancelCommand(args *CLIArgs, cfg *config.Config) error {
	client, err := createClient(args, cfg)
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}
	defer client.Close()

	task, err := client.CancelTask(context.Background(), args.AgentID, args.TaskID)
	if err != nil {
		return fmt.Errorf("failed to cancel task: %w", err)
	}

	fmt.Printf("✅ Task cancelled successfully\n\n")
	DisplayTask(task)
	return nil
}
