# hangar

a self-hosted deployment platform. declare your app in a `hangar.toml`, run `hangar deploy`, it lands on your vps with tls, a database, and a url.

the railway-shaped thing, but you own the box.

## why this exists

railway, render, fly, etc. go down, surprise-bill, or change pricing. self-hosting on a raw vps means writing your own docker + caddy + secrets + backups + firewall setup every time. coolify and dokploy try to solve this but are ui-first and heavy.

hangar's wedge: **cli-first, repo-declared, template-driven, self-hosted.** more like fly's ergonomics, runs on your own box.

## who it's for

andrew and alberto first. then anyone running chronalife/vaultly-shaped stuff who's tired of paas outages or surprise bills. technical users comfortable with a terminal.

## design rules

these are non-negotiable. every decision should be checked against them.

1. **cli mutates everything.** no web ui in v1. the ui is what killed hangar last time. status pages are auto-rendered html, no design work, no auth.
2. **the agent exposes one http api.** the cli is the only client in v1. never split the api across two backends.
3. **state lives on disk, updated atomically before any side effect.** no in-memory-only state. agent must survive crashes mid-deploy and resume.
4. **nothing user-visible changes until the caddy swap.** every failure before that is invisible to traffic.
5. **secrets are refs, never values, in `hangar.toml`.** values are set via cli, stored encrypted at rest on the agent.
6. **multi-tenant from day 1.** tenant is the unit of ownership. projects and standalone resources both live under a tenant.
7. **secure defaults, not "security platform."** hangar sets up a vps the way a competent person would set it up. that's the promise.

## architecture

```
+----------------------+         +-----------------------+
|   hangar cli (go)    | ------> | hangar agent (go)     |
|   (your laptop)      |  https  | (the vps, localhost   |
+----------------------+         |  + ssh tunnel only)   |
                                 |                       |
                                 |  docker daemon        |
                                 |  caddy admin api      |
                                 |  encrypted secrets    |
                                 |  postgres containers  |
                                 |  backup scheduler     |
                                 +-----------------------+
```

- **agent** — go daemon on the vps. one http api. accepts deploys, runs containers, manages caddy, owns databases, handles backups. listens on a unix socket or localhost only; cli connects via ssh tunnel.
- **cli** — go binary. shares types and toml schema with the agent via a common package.

## tech stack

| component | language | reason |
|-----------|----------|--------|
| agent | go | single binary, great docker/http story, boring on purpose |
| cli | go | shares code with agent, infra cli convention (cobra) |
| toml schema | go package | shared between cli and agent |

## the `hangar.toml`

one declarative file in the repo root. checked into git. no secrets.

```toml
schema = 1

[project]
name = "chronalife"          # slug; unique within tenant
region = "ash"               # v1: a label. multi-host comes later.

# ---- services ----
[[service]]
name = "api"
dockerfile = "./Dockerfile"
command = "bun run start"    # optional CMD override
port = 3000
replicas = 2

[service.healthcheck]
path = "/healthz"
interval = "10s"
timeout = "2s"
grace_period = "30s"

[service.resources]
cpu = "0.5"
memory = "512MB"

[service.http]
public = true
domains = ["api.chronalife.app"]
# auto subdomain always issued: chronalife-api.hngr.app

[[service]]
name = "worker"
dockerfile = "./Dockerfile"
command = "bun run worker"
replicas = 1
# no [service.http] = internal only, reachable from siblings at worker:3000

[service.resources]
cpu = "0.25"
memory = "256MB"

# ---- databases ----
[[database]]
name = "main"
engine = "postgres"
version = "16"
size = "10GB"

[[database]]
name = "cache"
engine = "redis"
version = "7"

# ---- env ----
# secrets and db urls are refs, never values
[env]
NODE_ENV     = "production"
LOG_LEVEL    = "info"
DATABASE_URL = "$db:main"
CACHE_URL    = "$db:cache"
SHARED_PG    = "$db:tenant/devpg"   # standalone db at tenant level
SENTRY_DSN   = "$secret:sentry_dsn"

# ---- deploy ----
[deploy]
strategy = "rolling"               # rolling | bluegreen | recreate
max_unavailable = 0
pre_deploy = "bun run db:migrate"
pre_deploy_service = "api"

# ---- volumes ----
# [[volume]]
# name = "uploads"
# service = "api"
# mount = "/data/uploads"
# size = "10GB"
```

### schema rules

- `schema = 1` is required and at the top. format can evolve without breaking old files.
- `[[service]]` is an array, not a map. services may differ in resources, healthchecks, public/internal, etc.
- tenant is **not** in the file. tenant is whoever's authed when `hangar deploy` runs.
- `name` is scoped to tenant, not global.
- secrets are refs only. inline values are rejected at parse time.
- `$db:name` references a project-local db. `$db:tenant/name` references a standalone tenant-level db.
- validation runs in two passes: toml syntax (library), then semantic (own code). semantic catches "service refs a db that doesn't exist," "port already claimed," etc.

## standalone databases

dbs can also exist outside a project, scoped to the tenant.

```
hangar db create devpg --engine postgres --version 16 --size 5GB
hangar db list
hangar db url devpg
hangar db connect devpg          # tunnels via agent, opens psql
hangar db expose devpg           # flip to public (with warning)
hangar db snapshot devpg
hangar db restore devpg --snapshot 2026-05-18
hangar db destroy devpg
```

defaults:
- created **private** (docker network only, reachable via `hangar db connect` tunnel)
- `hangar db expose` opens a high port + strong generated password, with a warning
- projects attach via `$db:tenant/name` in `hangar.toml`
- backups: nightly snapshots to local disk, optional s3/r2 offsite

## cli surface (v1)

```
hangar init                      # set up a vps (ssh hardening, firewall, agent install)
hangar deploy [--as-preview ID]  # deploy current dir, optional preview env
hangar logs [-f] [service]
hangar rollback
hangar status                    # project + host health
hangar secret set KEY VALUE
hangar secret list
hangar env list
hangar db create|list|url|connect|expose|snapshot|restore|destroy
hangar backup configure
```

## deploy flow

this is the spine. get it right or you'll be debugging forever.

```
1. cli: validate toml locally, resolve $secret: / $db: refs against agent
2. cli: tarball repo (respect .gitignore + .hangarignore), hash → build_id
3. cli: POST /deploys with toml + tarball + build_id → deploy_id
4. cli: stream /deploys/{id}/events over SSE, print to terminal
5. agent: write deploy dir to disk (toml, tarball, state.json) - atomic
6. agent: docker build with tag hangar/{tenant}/{project}/{service}:{build_id}
         - buildkit cache mount per project
         - fail = mark failed, nothing else changed
7. agent: run pre_deploy hook in one-shot container with full env
         - fail = abort, old version still serving, no rollback needed
8. agent: start new containers, named {project}-{service}-{build_id}-{n}
         - old containers untouched, still serving
9. agent: wait for healthchecks on new containers (respect grace_period)
         - fail within timeout = kill new containers, mark failed, old still serving
10. agent: PATCH caddy admin api to swap upstream pool atomically per service
          - in-flight requests on old containers finish naturally
11. agent: drain old containers (sigterm, wait, sigkill)
          - keep them stopped-but-present for fast rollback
12. agent: mark deploy succeeded, update current_build_id / previous_build_id
          - gc: keep last N successful builds, delete older containers + images
```

### rollback

```
hangar rollback
```

reverses step 10: caddy upstream back to old containers (restart them, 1-2s), drain new ones. if old containers got gc'd, fall back to redeploying previous build_id from cached image.

### crash recovery

every state transition writes `state.json` atomically. on agent restart, read every in-flight deploy dir and resume or mark failed based on the last recorded state:
- `building` → mark failed, no containers started
- `containers_started` (between step 9 and 10) → kill orphans, mark failed
- `swapped` (between step 10 and 11) → finish the drain
- `succeeded` / `failed` → nothing to do

## templates (post-v1)

a github org of pre-made `hangar.toml`'s for ghost, plausible, n8n, umami, etc. each template is a git repo with a `hangar.toml` + optional `template.toml` describing inputs.

```
hangar deploy --template ghost
```

clones the template, prompts for inputs (domain, admin email), deploys. contributions via pr to the templates org. you don't host anything.

## security defaults

`hangar init` sets all of this up automatically. opinionated by default, escape hatches via flags.

### ssh
- disable password auth (key-only)
- disable root login
- optional non-default port
- install + configure fail2ban for sshd

### firewall
- ufw deny by default
- allow ssh port, 80, 443
- agent api on unix socket / localhost only — no public surface

### updates
- unattended-upgrades for security patches only
- optional weekly maintenance window for reboots

### docker
- docker socket never exposed over tcp
- enforce non-root containers (warn loudly on root)
- per-project docker networks (projects can't reach each other)
- resource limits enforced from toml

### tls
- caddy auto-tls for public services
- internal traffic stays on docker networks

### secrets at rest
- encrypted store, key derived from passphrase set at `hangar init`
- unsealed into agent memory on start
- v1 acceptable fallback: file with 0400 perms, documented as "fix me before production"

### monitoring (surface via `hangar status`)
- failed ssh attempts (from fail2ban)
- containers restarting in a loop
- disk fill warning (#1 killer of small vps deploys)
- intermittent healthcheck failures
- caddy cert renewal failures
- stale auto-deploy projects (no push in N days)

### explicitly not doing in v1
- wireguard mesh between hosts (multi-host = different product)
- selinux/apparmor per-container profiles
- ossec/wazuh-style ids
- hashicorp vault integration
- compliance certs

## scope: v1 vs later

### v1 (just andrew + alberto, one host)
- agent (single host, single tenant — auth is a token in a config file)
- cli with full surface above
- `hangar.toml` parser + validator
- deploy flow end-to-end with caddy swap
- standalone databases (postgres, redis)
- secrets (file-based encrypted store)
- nightly local backups
- `hangar init` with secure defaults
- auto-rendered public status page per project

### later (in rough order)
- multi-tenant control plane with proper auth
- pr preview environments
- templates registry
- webhook-based deploys
- nixpacks-style builder for dockerfile-less repos
- offsite backups (s3/r2)
- multi-host scheduling
- canary / weighted rollouts

## build order

what to actually write first.

1. **caddy swap prototype.** 60-line script: start two nginx containers, swap caddy admin api between them, drain old. prove zero-downtime swap works reliably. this is the highest-risk piece.
2. **toml schema + validator in go.** the shared package. unit-test the bad cases (cyclic refs, unknown secrets, port conflicts).
3. **agent skeleton.** http api with the deploy/db/secret endpoints stubbed. state.json on disk. crash-resume logic from day 1.
4. **build + run pipeline.** docker build with cache, run container, wire to caddy from step 1. no healthchecks yet.
5. **healthchecks + the full deploy flow.** add the pre_deploy hook, healthcheck gating, drain.
6. **cli with `deploy`, `logs`, `rollback`.** ssh tunnel to agent. shared types with agent.
7. **secrets.** file-based encrypted store. cli commands.
8. **databases.** postgres + redis lifecycle. attach to projects via `$db:` refs.
9. **`hangar init`.** the vps setup script. ssh, firewall, agent install, fail2ban, unattended-upgrades.
10. **status pages.** auto-rendered html per project. read-only, public.
11. **backups.** local nightly snapshots first. offsite later.

## open questions to resolve as we go

- inter-service dns: per-project docker network, services reachable by name. confirm the exact docker compose semantics we're matching.
- build cache size cap and gc policy
- log retention defaults (disk + rotation)
- exact secret-store format (sqlite + sqlcipher? age-encrypted json? both have tradeoffs)
- how `hangar init` handles a host that's already partly configured (don't blow away existing ufw rules silently)
- exact auth model for v1: shared token in `~/.hangar/config` plus ssh? or just ssh and trust the unix socket?