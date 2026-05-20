package setup

import (
	"fmt"
	"net"
	"strconv"
	"strings"
)

type Target struct {
	User string
	Host string
	Port int
}

func ParseTarget(input string) (Target, error) {
	input = strings.TrimSpace(input)
	if input == "" {
		return Target{}, fmt.Errorf("expected user@host or host")
	}
	t := Target{User: "ubuntu", Port: 22}
	hostPart := input
	if i := strings.LastIndex(input, "@"); i >= 0 {
		t.User = input[:i]
		hostPart = input[i+1:]
		if t.User == "" {
			return Target{}, fmt.Errorf("missing user before @")
		}
	}
	if hostPart == "" {
		return Target{}, fmt.Errorf("missing host")
	}
	if strings.HasPrefix(hostPart, "[") {
		h, p, err := net.SplitHostPort(hostPart)
		if err != nil {
			return Target{}, fmt.Errorf("parse host:port: %w", err)
		}
		t.Host = h
		t.Port = atoiPort(p)
		return t, nil
	}
	if strings.Count(hostPart, ":") == 1 {
		h, p, err := net.SplitHostPort(hostPart)
		if err == nil {
			t.Host = h
			t.Port = atoiPort(p)
			return t, nil
		}
	}
	t.Host = hostPart
	return t, nil
}

func atoiPort(p string) int {
	n, err := strconv.Atoi(p)
	if err != nil || n <= 0 {
		return 22
	}
	return n
}

func (t Target) SSHAddr() string {
	if t.Port == 22 || t.Port == 0 {
		return t.User + "@" + t.Host
	}
	return fmt.Sprintf("%s@%s:%d", t.User, t.Host, t.Port)
}
