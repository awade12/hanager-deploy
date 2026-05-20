# Caddy swap prototype

Proves zero-downtime upstream swap via Caddy's admin API before the full deploy pipeline exists.

## Prerequisites

- Docker
- `curl`, `jq`

## Run

```bash
./scripts/caddy-swap/swap.sh
```

The script starts two nginx backends, routes traffic through Caddy, swaps upstreams, and drains the old pool.

## What it validates

1. Caddy admin API accepts JSON upstream patches
2. In-flight requests on the old upstream can finish after swap
3. Old upstream can be removed from the pool without dropping active connections
