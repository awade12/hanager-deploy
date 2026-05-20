package config

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

func TestSSH(user, host, keyPath string) error {
	return TestSSHPort(user, host, 22, keyPath)
}

func TestSSHPort(user, host string, port int, keyPath string) error {
	if _, err := os.Stat(keyPath); err != nil {
		return fmt.Errorf("ssh key %s: %w", keyPath, err)
	}

	try := func(batch bool) error {
		args := sshArgs(user, host, port, keyPath, batch)
		cmd := exec.Command("ssh", args...)
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Env = os.Environ()
		return cmd.Run()
	}

	if err := try(true); err == nil {
		return nil
	}

	fmt.Println("batch SSH failed, retrying (enter passphrase if prompted) ...")
	if err := try(false); err == nil {
		return nil
	}

	cmdLine := "ssh " + strings.Join(sshArgs(user, host, port, keyPath, true), " ")
	return fmt.Errorf(`ssh test failed

run this in your terminal:
  %s

if that fails, load your key:
  ssh-add --apple-use-keychain %s

or skip this check:
  hangar config init --skip-ssh-check --host %s --user %s --key-path %s`, cmdLine, keyPath, host, user, keyPath)
}

func sshArgs(user, host string, port int, keyPath string, batch bool) []string {
	args := []string{
		"-F", "/dev/null",
		"-i", keyPath,
		"-o", "IdentitiesOnly=yes",
		"-o", "PreferredAuthentications=publickey",
		"-o", "PasswordAuthentication=no",
		"-o", "ConnectTimeout=15",
		"-o", "StrictHostKeyChecking=accept-new",
	}
	if port != 0 && port != 22 {
		args = append(args, "-p", strconv.Itoa(port))
	}
	if batch {
		args = append(args, "-o", "BatchMode=yes")
	}
	args = append(args, user+"@"+host, "echo", "ok")
	return args
}
