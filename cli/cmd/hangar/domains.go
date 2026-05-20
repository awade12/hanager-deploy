package main

import (
	"encoding/json"
	"fmt"

	"github.com/awade12/hanager-deploy/cli/internal/bootstrap"
	"github.com/awade12/hanager-deploy/cli/internal/config"
	"github.com/spf13/cobra"
)

func domainsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "domains",
		Short: "public HTTPS domains on your VPS",
	}
	cmd.AddCommand(domainsEnableCmd())
	return cmd
}

func domainsEnableCmd() *cobra.Command {
	var email string
	cmd := &cobra.Command{
		Use:   "enable",
		Short: "enable public HTTPS (ports 80/443 + Let's Encrypt)",
		Long: `Turns on Caddy on ports 80/443 with automatic TLS.

1. Point your domain DNS A record to your VPS IP
2. Set domains in hangar.toml under [service.http]
3. Run hangar deploy`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if email == "" {
				return fmt.Errorf("--email is required for Let's Encrypt")
			}
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			agentCfg := map[string]any{
				"listen_addr":     "127.0.0.1:8741",
				"data_dir":        "/var/lib/hangar",
				"caddy_http_port": 8877,
				"caddy_admin_url": "http://127.0.0.1:2019",
				"caddy_container": "hangar-caddy",
				"public_edge":     true,
				"acme_email":      email,
			}
			if cfg.Token != "" {
				agentCfg["token"] = cfg.Token
			}
			data, err := json.MarshalIndent(agentCfg, "", "  ")
			if err != nil {
				return err
			}
			fmt.Println("==> enabling public HTTPS on VPS")
			if err := bootstrap.SSHWriteFile(cmd.Context(), cfg, "/etc/hangar/agent.json", string(data)); err != nil {
				return err
			}
			if err := bootstrap.SSHRun(cmd.Context(), cfg, "sudo docker rm -f hangar-caddy 2>/dev/null || true", ""); err != nil {
				return err
			}
			if err := bootstrap.SSHRun(cmd.Context(), cfg, "sudo systemctl restart hangar-agent", ""); err != nil {
				return err
			}
			fmt.Println()
			fmt.Printf("Public HTTPS enabled (ACME email: %s)\n", email)
			fmt.Println()
			fmt.Println("Next:")
			fmt.Printf("  1. DNS:  A record  your-domain.com  ->  %s\n", cfg.Host)
			fmt.Println("  2. hangar.toml:")
			fmt.Println("     [service.http]")
			fmt.Println("     public = true")
			fmt.Println("     domains = [\"your-domain.com\"]")
			fmt.Println("  3. hangar deploy")
			fmt.Println()
			fmt.Println("Then open https://your-domain.com")
			return nil
		},
	}
	cmd.Flags().StringVar(&email, "email", "", "email for Let's Encrypt (required)")
	_ = cmd.MarkFlagRequired("email")
	return cmd
}
