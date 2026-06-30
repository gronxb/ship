---
name: ship
description: Use when the user asks to deploy the current project with Ship, usually as "$ship 현재 프로젝트 demo로 배포 해줘" or "$ship deploy this Dockerfile project as <service>". Drives the local ship CLI safely with dry-run first, then deploys after the plan is valid.
---

# Ship

Deploy a Dockerfile project with the local `ship` CLI.

## Workflow

1. Confirm the target project directory contains a `Dockerfile`.
2. Derive the service name from the user request. If absent, use the current directory name normalized to DNS-safe lowercase.
3. Run the dry-run first:

```sh
ship --service <service> --cwd <project-dir> --dry-run --json
```

4. Read the JSON plan. Stop and report the exact error if planning fails.
5. If the plan is valid and the user asked to deploy, run:

```sh
ship --service <service> --cwd <project-dir>
```

6. Verify with the rollout command from the plan, then report the deployed host.

## Rules

- Do not invent domain, namespace, gateway, registry, or cluster values. Let `ship` read `~/.config/ship/config.env`, environment variables, and CLI defaults.
- Use `--env-file` only when the user names an env file or the project has a clear deployment env file such as `.env.production`.
- Use `--exposure internet` only when the user explicitly asks for public internet exposure.
- Keep deployment output concise: service, host, image, namespace, and verification command result.
