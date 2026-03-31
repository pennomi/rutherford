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
helm install rutherford ./chart \
  --set auth.oidc.issuer=https://accounts.google.com \
  --set auth.oidc.clientId=YOUR_CLIENT_ID
```

### With Ingress

```bash
helm install rutherford ./chart \
  --set auth.oidc.issuer=https://accounts.google.com \
  --set auth.oidc.clientId=YOUR_CLIENT_ID \
  --set ingress.enabled=true \
  --set ingress.hosts[0]=rutherford.example.com
```

## Configuration

### Auth (required)

| Value | Description |
|-------|-------------|
| `auth.provider` | Auth provider type (only `oidc` currently) |
| `auth.oidc.issuer` | OIDC issuer URL (e.g., `https://accounts.google.com`) |
| `auth.oidc.clientId` | OIDC client ID |
| `auth.oidc.audience` | JWT audience claim (defaults to clientId) |
| `auth.oidc.scopes` | OIDC scopes (default: `openid profile email`) |

### Storage scanning (optional)

For clusters using local-path-provisioner or similar host-based storage:

| Value | Description |
|-------|-------------|
| `storage.hostPath.enabled` | Enable host path scanning (default: false) |
| `storage.hostPath.path` | Host path to scan |
| `storage.hostPath.pattern` | Regex to extract PVC info from directory names |

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
