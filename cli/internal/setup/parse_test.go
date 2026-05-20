package setup

import "testing"

func TestParseTarget(t *testing.T) {
	cases := []struct {
		in       string
		user     string
		host     string
		port     int
		sshAddr  string
	}{
		{"ubuntu@15.204.234.121", "ubuntu", "15.204.234.121", 22, "ubuntu@15.204.234.121"},
		{"15.204.234.121", "ubuntu", "15.204.234.121", 22, "ubuntu@15.204.234.121"},
		{"root@example.com:2222", "root", "example.com", 2222, "root@example.com:2222"},
	}
	for _, c := range cases {
		got, err := ParseTarget(c.in)
		if err != nil {
			t.Fatalf("%q: %v", c.in, err)
		}
		if got.User != c.user || got.Host != c.host || got.Port != c.port || got.SSHAddr() != c.sshAddr {
			t.Fatalf("%q: got %+v", c.in, got)
		}
	}
}
