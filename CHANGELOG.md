# Changelog

All notable changes to Ship will be documented in this file.

The format follows Keep a Changelog and the project uses semantic versioning
once tagged releases begin.

## Unreleased

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
