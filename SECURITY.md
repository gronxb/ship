# Security Policy

## Supported Versions

Ship currently supports the `main` branch until tagged releases are introduced.

## Reporting A Vulnerability

Please report suspected vulnerabilities privately by opening a GitHub security
advisory for this repository. If that is not available, contact the maintainer
through the GitHub profile linked from the repository owner.

Include:

- affected commit or version
- reproduction steps
- expected impact
- whether credentials, cluster access, or DNS control are required

Do not include public exploit details until a fix is available.

## Security Model

Ship runs on a trusted operator machine and shells out to Docker, kubectl, kind,
and registry tooling. Anyone who can run Ship with your kubeconfig can deploy to
the configured namespace.

Recommended defaults:

- use tailnet-only exposure unless public routing is required
- review `ship --dry-run --json` output before first use on a cluster
- keep kubeconfig permissions scoped to the target namespaces
- rotate registry and DNS credentials if local operator machines are compromised
- keep `.env`, kubeconfigs, and generated secrets out of git
