package setup

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

const defaultKeyName = "hangar_deploy"

func DefaultKeyPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".ssh", defaultKeyName), nil
}

func EnsureKey(keyPath string) error {
	if _, err := os.Stat(keyPath); err == nil {
		fmt.Printf("using SSH key %s\n", keyPath)
		return nil
	} else if !os.IsNotExist(err) {
		return err
	}
	dir := filepath.Dir(keyPath)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	fmt.Printf("generating SSH key %s (no passphrase)\n", keyPath)
	cmd := exec.Command("ssh-keygen", "-t", "ed25519", "-f", keyPath, "-N", "", "-C", "hangar-deploy")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("ssh-keygen: %w", err)
	}
	return nil
}

func PublicKeyLine(keyPath string) (string, error) {
	data, err := os.ReadFile(keyPath + ".pub")
	if err != nil {
		return "", err
	}
	return string(data), nil
}
