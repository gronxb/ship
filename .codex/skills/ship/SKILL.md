---
name: ship
description: Use when the user asks to deploy or bring down a service with Ship, usually as "$ship deploy this project as demo" or "$ship bring down demo". Uses Docker Compose when present or prepares a Dockerfile, and previews deploy or teardown actions before applying them.
---

# Ship

Deploy or bring down a local service with the `ship` CLI.

## Workflow

1. Identify the target project directory.
2. Derive the service name from the user request. If absent, use the current directory name normalized to DNS-safe lowercase.
3. Prefer an existing `Dockerfile`. When no Dockerfile exists, use an existing canonical Compose file. Create a minimal Dockerfile only when neither source exists and the runtime can be inferred safely.
4. Run the dry-run first:

```sh
ship --service <service> --cwd <project-dir> --dry-run --json
```

5. Read the JSON plan. Stop and report the exact error if planning fails.
6. If the plan is valid and the user asked to deploy, run:

```sh
ship --service <service> --cwd <project-dir>
```

7. Verify Dockerfile deployments with the rollout command from the plan. Verify Compose deployments with `docker compose ps`, the managed Service/EndpointSlice/HTTPRoute, and the deployed host.

## Teardown Workflow

1. Identify the DNS-safe service name.
2. Preview the cleanup:

```sh
ship down --service <service> --dry-run
```

3. Read the plan and report any missing Kubernetes or Cloudflare prerequisite.
4. If the plan is valid and the user asked to bring the service down, run:

```sh
ship down --service <service>
```

5. Verify that the Deployment, Service, HTTPRoute, generated env Secret, legacy Ingress, and Compose EndpointSlice no longer exist. For Dockerfile services on local kind, also verify that the deployed image is absent from kind nodes and local Docker.

## Rules

- Prefer an existing `Dockerfile`; do not overwrite it unless the user explicitly asks.
- For Compose, prefer auto-detection and the `gateway` service. Use `--compose-file` or `--compose-service` when the project is non-canonical or ambiguous.
- Compose deployments currently require the configured local kind cluster. Do not attempt the host bridge against a remote Kubernetes context.
- Treat Compose files as trusted executable input. Do not render resolved Compose config or copy Compose env values into Kubernetes.
- Preserve Compose-owned project names, containers, and volumes. Never add `-p`, `down`, or volume deletion as an implicit deployment step.
- Treat `ship down` as destructive and always show its dry-run first.
- Preserve remote registry images and Compose-owned images. `ship down` removes Ship-managed local kind and Docker image copies only.
- Never bring down the configured Ship dashboard service individually; use `ship uninstall` only when the user explicitly asks to remove the whole Ship system.
- Keep generated Dockerfiles minimal and conventional. Use detected project commands instead of adding new dependencies.
- Do not invent domain, namespace, gateway, registry, or cluster values. Let `ship` read `~/.config/ship/config.env`, environment variables, and CLI defaults.
- Use `--env-file` only when the user names an env file or the project has a clear deployment env file such as `.env.production`.
- Use `--exposure internet` only when the user explicitly asks for public internet exposure.
- When redeploying an existing service, omit `--exposure` to preserve its current network path automatically.
- If you created a `Dockerfile`, mention that in the final report.
- Keep deployment output concise: service, host, image, namespace, and verification command result.
