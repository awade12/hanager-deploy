package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/awade12/hanager-deploy/cli/internal/config"
	"github.com/awade12/hanager-deploy/cli/internal/version"
	"github.com/spf13/cobra"
	"runtime/debug"
)

func configCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "manage ~/.hangar/config.json",
	}
	cmd.AddCommand(configInitCmd(), configShowCmd(), configPathCmd())
	return cmd
}

func configPathCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "path",
		Short: "print config file path",
		RunE: func(cmd *cobra.Command, args []string) error {
			path, err := config.Path()
			if err != nil {
				return err
			}
			fmt.Println(path)
			return nil
		},
	}
}

func configShowCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "show",
		Short: "print current config",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			path, err := config.Path()
			if err != nil {
				return err
			}
			fmt.Printf("# %s\n", path)
			fmt.Printf("host=%s user=%s port=%d tenant=%s\n", cfg.Host, cfg.User, cfg.Port, cfg.Tenant)
			fmt.Printf("key_path=%s local_port=%d remote_agent=%s\n", cfg.KeyPath, cfg.LocalPort, cfg.RemoteAgent)
			return nil
		},
	}
}

func configInitCmd() *cobra.Command {
	var host, user, keyPath, token string
	var skipSSHCheck bool
	cmd := &cobra.Command{
		Use:   "init",
		Short: "create ~/.hangar/config.json for laptop → VPS deploys",
		RunE: func(cmd *cobra.Command, args []string) error {
			reader := bufio.NewReader(os.Stdin)
			if host == "" {
				host = prompt(reader, "VPS hostname or IP: ")
			}
			if host == "" {
				return fmt.Errorf("host is required")
			}
			if user == "" {
				user = promptDefault(reader, "SSH user", "ubuntu")
			}
			if keyPath == "" {
				keyPath = promptDefault(reader, "SSH key path", "~/.ssh/id_ed25519")
			}
			var err error
			keyPath, err = config.ExpandKeyPath(keyPath)
			if err != nil {
				return err
			}

			if !skipSSHCheck {
				printConfigInitBuild()
				fmt.Printf("testing SSH to %s@%s ...\n", user, host)
				if err := config.TestSSH(user, host, keyPath); err != nil {
					return err
				}
				fmt.Println("SSH OK")
			}

			if token == "" {
				token = prompt(reader, "Agent token (optional): ")
			}

			cfg := config.Default()
			cfg.Host = host
			cfg.User = user
			cfg.KeyPath = keyPath
			cfg.Token = token
			path, err := config.Path()
			if err != nil {
				return err
			}
			if err := config.Save(cfg); err != nil {
				return err
			}
			fmt.Printf("wrote %s\n", path)
			fmt.Println("next: hangar init   (install agent on the VPS)")
			fmt.Println("then: hangar deploy (from any project with hangar.toml)")
			return nil
		},
	}
	cmd.Flags().StringVar(&host, "host", "", "VPS hostname or IP")
	cmd.Flags().StringVar(&user, "user", "", "SSH user")
	cmd.Flags().StringVar(&keyPath, "key-path", "", "SSH private key path")
	cmd.Flags().StringVar(&token, "token", "", "agent bearer token")
	cmd.Flags().BoolVar(&skipSSHCheck, "skip-ssh-check", false, "write config without testing SSH")
	return cmd
}

func prompt(reader *bufio.Reader, label string) string {
	fmt.Print(label)
	line, _ := reader.ReadString('\n')
	return strings.TrimSpace(line)
}

func promptDefault(reader *bufio.Reader, label, def string) string {
	fmt.Printf("%s [%s]: ", label, def)
	line, _ := reader.ReadString('\n')
	line = strings.TrimSpace(line)
	if line == "" {
		return def
	}
	return line
}

func printConfigInitBuild() {
	if info, ok := debug.ReadBuildInfo(); ok && info.Main.Version != "" && info.Main.Version != "(devel)" {
		fmt.Printf("(%s %s)\n", version.String(), info.Main.Version)
		return
	}
	fmt.Printf("(%s — run: go install %s/cli/cmd/hangar@latest)\n", version.String(), version.Module())
}
