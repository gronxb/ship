---
name: ship
description: Use when the user asks to deploy the current project with Ship, usually as "$ship deploy this project as <service>". Prepares the project, creates a Dockerfile when needed, runs a dry-run first, then deploys after the plan is valid.
---

# Ship

Deploy a local project with the `ship` CLI.

## Workflow

1. Identify the target project directory.
2. Derive the service name from the user request. If absent, use the current directory name normalized to DNS-safe lowercase.
3. If the project has no `Dockerfile`, inspect the project and create a minimal suitable one using its existing runtime, lockfile, package manager, build script, and start command. If the runtime cannot be inferred safely, stop and report the missing information.
4. Run the dry-run first:

```sh
ship --service <service> --cwd <project-dir> --dry-run --json
```

5. Read the JSON plan. Stop and report the exact error if planning fails.
6. If the plan is valid and the user asked to deploy, run:

```sh
ship --service <service> --cwd <project-dir>
```

7. Verify with the rollout command from the plan, then report the deployed host.

## Rules

- Prefer an existing `Dockerfile`; do not overwrite it unless the user explicitly asks.
- Keep generated Dockerfiles minimal and conventional. Use detected project commands instead of adding new dependencies.
- Do not invent domain, namespace, gateway, registry, or cluster values. Let `ship` read `~/.config/ship/config.env`, environment variables, and CLI defaults.
- Use `--env-file` only when the user names an env file or the project has a clear deployment env file such as `.env.production`.
- Use `--exposure internet` only when the user explicitly asks for public internet exposure.
- If you created a `Dockerfile`, mention that in the final report.
- Keep deployment output concise: service, host, image, namespace, and verification command result.
