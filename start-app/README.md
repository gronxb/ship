# Ship Dashboard

Dashboard for Ship deployments. It lists deployed containers as cards and
exposes network requests, terminal commands, rendered manifests, recent
container logs, and route exposure controls.

```sh
pnpm install
pnpm dev --host 0.0.0.0 --port 3000
```

The API reads the same Ship config as the CLI:

```sh
~/.config/ship/config.env
```

Deploy it as the `k8s` service, then open `https://k8s.mydomain.com` from your
tailnet:

```sh
cd ../deploy-system
./deploy-dashboard.sh
```

## Verification

```sh
pnpm test
pnpm typecheck
pnpm lint
pnpm build
```

The production image runs as an unprivileged `ship` user.
