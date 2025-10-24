package cli

import (
	"context"
	"fmt"

	"github.com/kadirpekel/hector/pkg/config"
)

func TaskGetCommand(args *TaskGetCmd, cfg *config.Config, mode CLIMode) error {
	client, err := createClient(args, cfg, mode)
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}
	defer client.Close()

	task, err := client.GetTask(context.Background(), args.Agent, args.TaskID)
	if err != nil {
		return fmt.Errorf("failed to get task: %w", err)
	}

	DisplayTask(task)
	return nil
}

func TaskCancelCommand(args *TaskCancelCmd, cfg *config.Config, mode CLIMode) error {
	client, err := createClient(args, cfg, mode)
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}
	defer client.Close()

	task, err := client.CancelTask(context.Background(), args.Agent, args.TaskID)
	if err != nil {
		return fmt.Errorf("failed to cancel task: %w", err)
	}

	fmt.Printf("✅ Task cancelled successfully\n\n")
	DisplayTask(task)
	return nil
}
