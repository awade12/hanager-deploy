package schema

type Manifest struct {
	Schema   int              `toml:"schema"`
	Project  Project          `toml:"project"`
	Services []Service        `toml:"service"`
	Databases []Database      `toml:"database"`
	Env      map[string]string `toml:"env"`
	Deploy   Deploy           `toml:"deploy"`
}

type Project struct {
	Name   string `toml:"name"`
	Region string `toml:"region"`
}

type Service struct {
	Name        string       `toml:"name"`
	Dockerfile  string       `toml:"dockerfile"`
	Command     string       `toml:"command"`
	Port        int          `toml:"port"`
	Replicas    int          `toml:"replicas"`
	Healthcheck *Healthcheck `toml:"healthcheck"`
	Resources   *Resources   `toml:"resources"`
	HTTP        *HTTP        `toml:"http"`
}

type Healthcheck struct {
	Path        string `toml:"path"`
	Interval    string `toml:"interval"`
	Timeout     string `toml:"timeout"`
	GracePeriod string `toml:"grace_period"`
}

type Resources struct {
	CPU    string `toml:"cpu"`
	Memory string `toml:"memory"`
}

type HTTP struct {
	Public  bool     `toml:"public"`
	Domains []string `toml:"domains"`
}

type Database struct {
	Name    string `toml:"name"`
	Engine  string `toml:"engine"`
	Version string `toml:"version"`
	Size    string `toml:"size"`
}

type Deploy struct {
	Strategy           string `toml:"strategy"`
	MaxUnavailable     int    `toml:"max_unavailable"`
	PreDeploy          string `toml:"pre_deploy"`
	PreDeployService   string `toml:"pre_deploy_service"`
}
