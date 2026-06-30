# Ship Dashboard

Read-only dashboard for Ship deployments. It lists deployed containers as cards
and exposes network requests, terminal commands, rendered manifests, and recent
container logs.

```sh
pnpm install
pnpm dev --host 0.0.0.0 --port 9292
```

The API reads the same Ship config as the CLI:

```sh
~/.config/ship/config.env
```

Deploy the dashboard route from `../deploy-system`, then open
`https://k8s.mydomain.com` from your tailnet.

## Verification

```sh
pnpm test
pnpm typecheck
pnpm lint
pnpm build
```

The production image runs as an unprivileged `ship` user.
