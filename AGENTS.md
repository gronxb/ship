# Repository Agent Instructions

## Development Refresh

- If changes to `cmd/ship`, `internal/deploy`, `deploy-system`, or `start-app`
  need to show up in local development, run `make dev-refresh`.
- `make dev-refresh` rebuilds the current checkout into
  `${SHIP_BIN:-$HOME/.local/bin/ship}` and redeploys the dashboard through the
  existing `deploy-system/deploy-dashboard.sh` path.
- This is a repository developer command, not a Ship CLI feature. Do not add a
  `ship dev-refresh` subcommand unless the user explicitly asks for product
  behavior.
- The command expects the normal Ship development prerequisites: Go, Docker,
  kubectl access, kind or `REGISTRY`, and the repo `.env` or Ship config used by
  `deploy-system/ship-env.sh`.
