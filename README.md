# hangar

Self-hosted deployment platform. Repo: [github.com/awade12/hanager-deploy](https://github.com/awade12/hanager-deploy). See [project.md](./project.md).

## Install (any machine — no repo needed)

**With Go 1.22+:**

```bash
go install github.com/awade12/hanager-deploy/cli/cmd/hangar@v0.2.0
go install github.com/awade12/hanager-deploy/agent/cmd/hangar-agent@v0.2.0
```

If `@latest` installs an old build, use `@v0.1.0` or:

```bash
GOPROXY=direct go install github.com/awade12/hanager-deploy/cli/cmd/hangar@main
```

Ensure `$(go env GOPATH)/bin` is on your `PATH`.

**One-liner installer** (Go or GitHub release binaries):

```bash
curl -fsSL "https://raw.githubusercontent.com/awade12/hanager-deploy/main/scripts/install.sh?$(date +%s)" | bash
```

You should see `hangar installer v2` first. If you still see `hangar-sh` or `BASH_SOURCE` errors, your shell cached an old script — use the `go install` commands below instead.

**One command setup** (recommended):

```bash
hangar setup ubuntu@YOUR_VPS_IP
cd ~/code/my-app
hangar deploy
```

`setup` creates `~/.ssh/hangar_deploy`, copies it to the server (`ssh-copy-id`), writes `~/.hangar/config.json`, and installs the agent.

**Manual workflow** (if you prefer):

```bash
hangar config init --skip-ssh-check ...
hangar init
hangar deploy
```

`hangar init` finds `hangar-agent` on your PATH, or runs `go install` for you if Go is available.

You only need the hangar repo if you are **developing** hangar itself.

## Layout

```
agent/          # VPS daemon (HTTP API, deploy state, recovery)
cli/            # User-facing CLI
pkg/schema/     # Shared hangar.toml parse + validate
pkg/fsutil/     # Atomic JSON writes
scripts/        # install.sh, deploy spikes
```

## Develop (from this repo)

Prerequisites: **Go 1.22+**, **Docker**.

```bash
make install-go    # ubuntu only, if system go is too old
make setup         # build bin/hangar and bin/hangar-agent
make test
```

Local agent:

```bash
make stop-agent
./bin/hangar-agent -config ./agent.dev.json
make deploy-hello
```

Agent config example (`agent.dev.json`):

```json
{
  "listen_addr": "127.0.0.1:8741",
  "data_dir": "/var/lib/hangar",
  "token": "dev-token",
  "caddy_admin_url": "http://127.0.0.1:2019",
  "caddy_http_port": 8877,
  "caddy_container": "hangar-caddy"
}
```

Hangar Caddy listens on **127.0.0.1:8877** by default.

### Secrets and databases

- Secrets: AES-encrypted at `{data_dir}/secrets/{tenant}.enc`, key in `secrets.key` (mode 0600).
- `$secret:name` and `$db:name` / `$db:tenant/name` are resolved at deploy time.
- `[[database]]` in `hangar.toml` are provisioned on the project docker network during deploy.

After upgrading the agent, recreate Caddy once so routes work:

```bash
docker rm -f hangar-caddy
```

### Try a hello deploy

Requires Docker. Start the agent with a writable `data_dir`, then:

```bash
./bin/hangar-agent -config ./agent.dev.json
./scripts/deploy-hello.sh
```

## Caddy swap spike

```bash
make caddy-swap
```

Requires Docker.
