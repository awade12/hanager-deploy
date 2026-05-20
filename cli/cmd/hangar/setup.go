package main

import (
	"github.com/awade12/hanager-deploy/cli/internal/setup"
	"github.com/spf13/cobra"
)

func setupCmd() *cobra.Command {
	var keyPath, dataDir string
	cmd := &cobra.Command{
		Use:   "setup [user@host]",
		Short: "one-shot: SSH key, config, and agent install on your VPS",
		Long: `Example:
  hangar setup ubuntu@15.204.234.121

Creates ~/.ssh/hangar_deploy if needed, copies it to the server,
writes ~/.hangar/config.json, and installs hangar-agent.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			target, err := setup.ParseTarget(args[0])
			if err != nil {
				return err
			}
			return setup.Run(cmd.Context(), target, setup.Options{
				KeyPath: keyPath,
				DataDir: dataDir,
			})
		},
	}
	cmd.Flags().StringVar(&keyPath, "key-path", "", "SSH private key (default: ~/.ssh/hangar_deploy)")
	cmd.Flags().StringVar(&dataDir, "data-dir", "/var/lib/hangar", "agent data directory on VPS")
	return cmd
}
