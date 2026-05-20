package caddy

import (
	"path/filepath"
	"strconv"
)

type EdgeMode struct {
	Public    bool
	ACMEEmail string
	LocalPort int
}

func (m EdgeMode) ListenAddrs() []string {
	if m.Public {
		return []string{":80", ":443"}
	}
	port := m.LocalPort
	if port == 0 {
		port = 8877
	}
	return []string{":" + strconv.Itoa(port)}
}

func (m EdgeMode) DockerPorts() []string {
	if m.Public {
		return []string{
			"0.0.0.0:80:80",
			"0.0.0.0:443:443",
			"127.0.0.1:2019:2019",
		}
	}
	port := m.LocalPort
	if port == 0 {
		port = 8877
	}
	return []string{
		"127.0.0.1:2019:2019",
		"127.0.0.1:" + strconv.Itoa(port) + ":" + strconv.Itoa(port),
	}
}

func (m EdgeMode) CaddyfileGlobal() string {
	if m.Public && m.ACMEEmail != "" {
		return "{\n\tadmin 0.0.0.0:2019\n\temail " + m.ACMEEmail + "\n}\n"
	}
	return "{\n\tadmin 0.0.0.0:2019\n}\n"
}

func (m EdgeMode) ExtraMounts(caddyDir string) []string {
	if !m.Public {
		return nil
	}
	return []string{
		filepath.Join(caddyDir, "data") + ":/data",
		filepath.Join(caddyDir, "config") + ":/config",
	}
}
