# Changelog

All notable changes to Ship will be documented in this file.

The format follows Keep a Changelog and the project uses semantic versioning
once tagged releases begin.

## Unreleased
## 0.1.9 - 2026-07-12

### Fixed

- Preserve the current network exposure on redeploys: services already routed
  through the internet stay on the internet path, and Tailscale services stay
  Tailscale-only unless `--exposure` is passed explicitly.
- Reuse existing HTTPRoute exposure labels during redeploy preflight, including
  legacy `ship.local/tailscale-only=true` routes.

### Docs

- Document automatic exposure preservation for redeploys in the README and Ship
  agent skill guidance.

## 0.1.8 - 2026-07-12

### Added

- Native local Docker Compose deployments backed by a selectorless Kubernetes
  Service, EndpointSlice, and HTTPRoute, with dry-run planning and readiness
  checks that keep Compose environment values out of Kubernetes.

### Security

- Validate Kubernetes and DNS manifest inputs, sanitize Compose failures, and
  shell-quote dynamic Dockerfile dry-run command arguments.

## 0.1.7 - 2026-07-09

### Added

- Read-only dashboard for deployed container cards, network requests, terminal
  commands, manifests, and container logs.
- Release-readiness checks, GitHub CI, issue templates, and contribution docs.
- Tailnet-first deploy-system manifests with optional internet Gateway support.
