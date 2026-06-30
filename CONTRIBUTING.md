# Contributing

Thanks for helping improve Ship. The project should stay small, direct, and
auditable.

## Development Setup

Install the required tools:

- Go 1.22 or newer
- Docker
- kubectl
- pnpm 10 or newer

Install dashboard dependencies:

```sh
cd start-app
pnpm install
```

Run the full verification suite from the repository root:

```sh
make test
```

## Pull Request Checklist

- Keep changes focused and reversible.
- Include tests for behavior changes.
- Run `make test`.
- Update docs when behavior, flags, config, or deployment assumptions change.
- Do not commit secrets, kubeconfigs, local `.env` files, build output, or local
  OMX evidence.

## Design Principles

- Prefer explicit Kubernetes resources over hidden platform behavior.
- Keep the browser dashboard read-only.
- Default to tailnet-only exposure.
- Make public internet exposure opt-in and easy to review.
- Keep install and recovery paths shell-native.
