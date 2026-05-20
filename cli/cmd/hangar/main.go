package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/awade12/hanager-deploy/cli/internal/agentclient"
	"github.com/awade12/hanager-deploy/cli/internal/bootstrap"
	"github.com/awade12/hanager-deploy/cli/internal/config"
	"github.com/awade12/hanager-deploy/cli/internal/version"
	"github.com/awade12/hanager-deploy/pkg/archive"
	"github.com/awade12/hanager-deploy/pkg/schema"
	"github.com/spf13/cobra"
)

func main() {
	if err := newRoot().Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func newRoot() *cobra.Command {
	root := &cobra.Command{
		Use:   "hangar",
		Short: "hangar deployment cli",
	}
	root.AddCommand(
		deployCmd(),
		logsCmd(),
		rollbackCmd(),
		secretCmd(),
		dbCmd(),
		initCmd(),
		statusCmd(),
		configCmd(),
		versionCmd(),
	)
	return root
}

func versionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "print version",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println(version.String())
		},
	}
}

func deployCmd() *cobra.Command {
	var dir string
	cmd := &cobra.Command{
		Use:   "deploy",
		Short: "deploy current project to the agent",
		RunE: func(cmd *cobra.Command, args []string) error {
			if dir == "" {
				dir = "."
			}
			dir, err := filepath.Abs(dir)
			if err != nil {
				return err
			}
			tomlPath := filepath.Join(dir, "hangar.toml")
			tomlBytes, err := os.ReadFile(tomlPath)
			if err != nil {
				return fmt.Errorf("read hangar.toml: %w", err)
			}
			manifest, err := schema.Parse(tomlBytes)
			if err != nil {
				return err
			}
			return withAgent(cmd.Context(), func(client *agentclient.Client, cfg config.Config) error {
				if err := validateRefs(cmd, client, cfg, manifest); err != nil {
					return err
				}
				tmp := filepath.Join(os.TempDir(), fmt.Sprintf("hangar-src-%d.tar.gz", time.Now().UnixNano()))
				defer os.Remove(tmp)
				if err := archive.PackDirToFile(dir, tmp, ".hangarignore"); err != nil {
					return fmt.Errorf("pack source: %w", err)
				}
				sum := sha256.Sum256(tomlBytes)
				buildID := "build-" + hex.EncodeToString(sum[:6])
				st, err := client.Deploy(cmd.Context(), cfg.Tenant, buildID, tomlPath, tmp)
				if err != nil {
					return err
				}
				fmt.Printf("deploy_id=%s project=%s build_id=%s\n", st.ID, manifest.Project.Name, buildID)
				return client.StreamEvents(cmd.Context(), st.ID, func(s agentclient.DeployState) {
					fmt.Printf("[%s] %s\n", s.Phase, s.Message)
				})
			})
		},
	}
	cmd.Flags().StringVar(&dir, "dir", ".", "project directory")
	return cmd
}

func validateRefs(cmd *cobra.Command, client *agentclient.Client, cfg config.Config, m *schema.Manifest) error {
	keys, err := client.SecretList(cmd.Context(), cfg.Tenant)
	if err != nil {
		return err
	}
	keySet := map[string]bool{}
	for _, k := range keys {
		keySet[k] = true
	}
	dbs, err := client.DBList(cmd.Context(), cfg.Tenant)
	if err != nil {
		return err
	}
	dbSet := map[string]bool{}
	for _, d := range dbs {
		dbSet[d] = true
	}
	for k, v := range m.Env {
		ref, ok := schema.ParseEnvValue(v)
		if !ok {
			continue
		}
		switch ref.Kind {
		case schema.RefSecret:
			if !keySet[ref.SecretName] {
				return fmt.Errorf("env.%s: secret %q not found (hangar secret set %s)", k, ref.SecretName, ref.SecretName)
			}
		case schema.RefDB:
			if ref.TenantDB {
				if !dbSet[ref.DBName] {
					return fmt.Errorf("env.%s: tenant database %q not found (hangar db create %s)", k, ref.DBName, ref.DBName)
				}
				continue
			}
			found := false
			for _, d := range m.Databases {
				if d.Name == ref.DBName {
					found = true
					break
				}
			}
			if !found && !dbSet[ref.DBName] {
				return fmt.Errorf("env.%s: database %q not in hangar.toml [[database]]", k, ref.DBName)
			}
		}
	}
	return nil
}

func logsCmd() *cobra.Command {
	var serviceName, tail string
	var follow bool
	cmd := &cobra.Command{
		Use:   "logs [service]",
		Short: "fetch service logs from the agent",
		RunE: func(cmd *cobra.Command, args []string) error {
			svc := serviceName
			if len(args) > 0 {
				svc = args[0]
			}
			return withAgent(cmd.Context(), func(client *agentclient.Client, cfg config.Config) error {
				manifest, err := schema.ParseFile("hangar.toml")
				if err != nil {
					return fmt.Errorf("run from project root: %w", err)
				}
				if follow {
					for {
						out, err := client.Logs(cmd.Context(), cfg.Tenant, manifest.Project.Name, svc, tail)
						if err != nil {
							return err
						}
						fmt.Print(out)
						time.Sleep(2 * time.Second)
					}
				}
				out, err := client.Logs(cmd.Context(), cfg.Tenant, manifest.Project.Name, svc, tail)
				if err != nil {
					return err
				}
				fmt.Print(out)
				return nil
			})
		},
	}
	cmd.Flags().StringVar(&serviceName, "service", "", "service name")
	cmd.Flags().StringVar(&tail, "tail", "100", "lines of logs")
	cmd.Flags().BoolVarP(&follow, "follow", "f", false, "poll logs every 2s")
	return cmd
}

func rollbackCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "rollback",
		Short: "roll back to the previous deployment",
		RunE: func(cmd *cobra.Command, args []string) error {
			return withAgent(cmd.Context(), func(client *agentclient.Client, cfg config.Config) error {
				manifest, err := schema.ParseFile("hangar.toml")
				if err != nil {
					return err
				}
				if err := client.Rollback(cmd.Context(), cfg.Tenant, manifest.Project.Name); err != nil {
					return err
				}
				fmt.Println("rollback complete")
				return nil
			})
		},
	}
}

func secretCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "secret", Short: "manage secrets on the agent"}
	cmd.AddCommand(
		&cobra.Command{
			Use:   "set KEY VALUE",
			Short: "set a secret",
			Args:  cobra.ExactArgs(2),
			RunE: func(cmd *cobra.Command, args []string) error {
				return withAgent(cmd.Context(), func(client *agentclient.Client, cfg config.Config) error {
					if err := client.SecretSet(cmd.Context(), cfg.Tenant, args[0], args[1]); err != nil {
						return err
					}
					fmt.Printf("secret %q set\n", args[0])
					return nil
				})
			},
		},
		&cobra.Command{
			Use:   "list",
			Short: "list secret names",
			RunE: func(cmd *cobra.Command, args []string) error {
				return withAgent(cmd.Context(), func(client *agentclient.Client, cfg config.Config) error {
					keys, err := client.SecretList(cmd.Context(), cfg.Tenant)
					if err != nil {
						return err
					}
					for _, k := range keys {
						fmt.Println(k)
					}
					return nil
				})
			},
		},
	)
	return cmd
}

func dbCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "db", Short: "manage tenant databases"}
	create := &cobra.Command{
		Use:   "create NAME",
		Short: "create a standalone database",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			engine, _ := cmd.Flags().GetString("engine")
			version, _ := cmd.Flags().GetString("version")
			return withAgent(cmd.Context(), func(client *agentclient.Client, cfg config.Config) error {
				if err := client.DBCreate(cmd.Context(), cfg.Tenant, args[0], engine, version); err != nil {
					return err
				}
				fmt.Printf("database %q created\n", args[0])
				return nil
			})
		},
	}
	create.Flags().String("engine", "postgres", "postgres or redis")
	create.Flags().String("version", "", "image version")
	cmd.AddCommand(
		create,
		&cobra.Command{
			Use:   "list",
			Short: "list databases",
			RunE: func(cmd *cobra.Command, args []string) error {
				return withAgent(cmd.Context(), func(client *agentclient.Client, cfg config.Config) error {
					names, err := client.DBList(cmd.Context(), cfg.Tenant)
					if err != nil {
						return err
					}
					for _, n := range names {
						fmt.Println(n)
					}
					return nil
				})
			},
		},
		&cobra.Command{
			Use:   "url NAME",
			Short: "print database connection url",
			Args:  cobra.ExactArgs(1),
			RunE: func(cmd *cobra.Command, args []string) error {
				return withAgent(cmd.Context(), func(client *agentclient.Client, cfg config.Config) error {
					u, err := client.DBURL(cmd.Context(), cfg.Tenant, args[0])
					if err != nil {
						return err
					}
					fmt.Println(u)
					return nil
				})
			},
		},
	)
	return cmd
}

func initCmd() *cobra.Command {
	var agentBin, dataDir string
	cmd := &cobra.Command{
		Use:   "init",
		Short: "bootstrap a fresh vps (docker, firewall, hangar-agent)",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			agentPath, err := bootstrap.ResolveAgentBinary(cmd.Context(), agentBin)
			if err != nil {
				return err
			}
			return bootstrap.Run(cmd.Context(), cfg, bootstrap.Options{
				AgentBinary: agentPath,
				DataDir:     dataDir,
			})
		},
	}
	cmd.Flags().StringVar(&agentBin, "agent-bin", "", "path to hangar-agent (default: PATH or go install)")
	cmd.Flags().StringVar(&dataDir, "data-dir", "/var/lib/hangar", "agent data directory on vps")
	return cmd
}

func statusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "show project runtime on the agent",
		RunE: func(cmd *cobra.Command, args []string) error {
			return withAgent(cmd.Context(), func(client *agentclient.Client, cfg config.Config) error {
				manifest, err := schema.ParseFile("hangar.toml")
				if err != nil {
					return err
				}
				body, err := client.GetProject(cmd.Context(), cfg.Tenant, manifest.Project.Name)
				if err != nil {
					return err
				}
				fmt.Println(body)
				return nil
			})
		},
	}
}
