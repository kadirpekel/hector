package cli

import (
	"context"
	"fmt"

	"github.com/kadirpekel/hector/pkg/config"
	"github.com/kadirpekel/hector/pkg/server"
)

func ServeCommand(args *ServeCmd, cfg *config.Config, mode CLIMode) error {

	srv, err := server.New(server.Options{
		Config:  cfg,
		Host:    args.Host,
		Port:    args.Port,
		BaseURL: args.A2ABaseURL,
		Debug:   CLI.Debug,
	})
	if err != nil {
		return fmt.Errorf("server creation failed: %w", err)
	}

	if err := srv.Start(context.Background()); err != nil {
		return fmt.Errorf("server startup failed: %w", err)
	}

	srv.Wait()

	return nil
}
