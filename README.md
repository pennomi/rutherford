# Rutherford

Real-time Kubernetes monitoring dashboard. Streams cluster state over WebSocket — no polling, no refresh. Single binary with enforced OIDC authentication.

## Features

- Real-time updates via WebSocket (K8s watch API -> server -> browser)
- Monitors 24+ resource types: pods, deployments, statefulsets, services, ingresses, RBAC, and more
- Live metrics: CPU, memory, disk usage with historical charts
- Pod log streaming
- Hot resource detection (containers near CPU/memory limits)
- Namespace-level health overview
- Single Go binary with embedded SvelteKit SPA
- Enforced OIDC authentication (Google, Keycloak, Dex, Zitadel, Auth0, Okta, etc.)

## Quick Start

### Prerequisites

- Kubernetes cluster with [metrics-server](https://github.com/kubernetes-sigs/metrics-server) installed
- An OIDC provider (Google, Keycloak, etc.) with a client ID configured

### Install with Helm

```bash
helm install rutherford oci://ghcr.io/pennomi/rutherford --version 0.1.0 \
  --set auth.oidc.issuer=YOUR_OIDC_ISSUER_URL \
  --set auth.oidc.clientId=YOUR_OIDC_CLIENT_ID \
  --set auth.oidc.audience=YOUR_OIDC_AUDIENCE \
  --set "auth.oidc.allowedEmails={alice@company.com,bob@company.com}" \
  --set "auth.oidc.allowedGroups={k8s-admins}" \
  --set "ingress.hosts={YOUR_HOSTNAME_1,YOUR_HOSTNAME_2}"
```

## Configuration

| Value                  | Description                        |
| ---------------------- | ---------------------------------- |
| `auth.oidc.issuer`    | OIDC issuer URL                    |
| `auth.oidc.clientId`  | OIDC client ID                     |
| `auth.oidc.audience`  | JWT audience claim                 |
| `auth.oidc.allowedEmails` | Whitelisted email addresses    |
| `auth.oidc.allowedGroups` | Whitelisted group names        |
| `ingress.hosts`       | List of hostnames                  |
| `ingress.tls`         | TLS configuration                  |
| `ingress.annotations` | Ingress annotations                |
| `image.tag`           | Image tag (default: `latest`)      |

### Namespace icons

Set the `rutherford/icon` annotation on a namespace to display a custom icon:

```yaml
apiVersion: v1
kind: Namespace
metadata:
  name: my-app
  annotations:
    rutherford/icon: "https://example.com/icon.png"
```

## Development

Run the Go API and SvelteKit dev server separately:

```bash
# Terminal 1: API server (requires in-cluster config or kubeconfig)
go run .

# Terminal 2: UI dev server with hot reload
cd ui && npm run dev
```

The Vite dev server proxies `/api/*` and `/ws/*` to the Go server on `:8080`.

## Architecture

```
Browser <--WebSocket--> Go Server <--Watch API--> Kubernetes API
                        |
                        +-- Embedded SvelteKit SPA (served as static files)
                        +-- OIDC JWT validation (JWKS discovery)
                        +-- Metrics polling (15s interval)
```

## License

[AGPL-3.0](LICENSE)
