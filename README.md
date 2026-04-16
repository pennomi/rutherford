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
- An OIDC provider with a client configured for PKCE (public client, authorization code + PKCE)

### Install with Kustomize

Two overlay examples are provided under `kustomize/examples/`:

- `google/` — OAuth 2.0 Client ID downloaded from Google Cloud Console
- `oidc/` — generic OIDC provider with a simple `{ issuer, clientId }` JSON

Copy one, fill in your credentials, and apply:

```bash
cp -r kustomize/examples/oidc my-rutherford

# Drop in your auth config:
cp my-rutherford/auth.json.example my-rutherford/auth.json
$EDITOR my-rutherford/auth.json

# Set the allowed email list and hostname:
$EDITOR my-rutherford/kustomization.yaml
$EDITOR my-rutherford/ingress.yaml

kubectl apply -k my-rutherford
```

For Google, download the OAuth Client JSON from the GCP console and save it as
`client_secret.json` in the overlay directory — Rutherford detects the `web`
wrapper automatically.

## Auth Configuration

Rutherford reads a single JSON file (mounted as a Kubernetes secret at
`/etc/rutherford/auth.json` by default) and auto-detects the format:

**Google** (the unmodified JSON from Google Cloud Console):

```json
{
  "web": {
    "client_id": "...apps.googleusercontent.com",
    "...": "other fields are ignored"
  }
}
```

**Generic OIDC / PKCE**:

```json
{
  "issuer": "https://auth.example.com/realms/my-realm",
  "clientId": "rutherford",
  "scopes": "openid profile email"
}
```

`scopes` is optional and defaults to `openid profile email`. Any parse or
discovery error causes Rutherford to panic on startup — there are no
fallbacks.

### Allow list

Set `ALLOWED_EMAILS` as an environment variable (comma-separated) to restrict
access. In the provided overlays it is sourced from the same secret as
`auth.json`:

```yaml
secretGenerator:
  - name: rutherford-auth
    files:
      - auth.json
    literals:
      - allowed_emails=alice@example.com,bob@example.com
```

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
go run . --kubeconfig ~/.kube/config --no-auth

# Terminal 2: UI dev server with hot reload
cd ui && npm run dev
```

The Vite dev server proxies `/api/*` and `/ws/*` to the Go server on `:8080`.

### Testing auth locally

`--no-auth` skips the whole auth path. To exercise real OIDC in dev, drop
`--no-auth` and point `--auth-config` at a local JSON file:

1. Register `http://localhost:5173/callback` as an allowed redirect URI with
   your OIDC provider (Vite's default dev port). For Google, also add
   `http://localhost:5173` under "Authorized JavaScript origins".
2. Save the provider's JSON somewhere git-ignored (e.g.
   `~/.config/rutherford-dev.json`) — either the raw Google client JSON or the
   simple `{ "issuer": "...", "clientId": "..." }` form.
3. Run:

   ```bash
   # Terminal 1
   go run . \
     --kubeconfig ~/.kube/config \
     --auth-config ~/.config/rutherford-dev.json \
     --port 8081

   # Terminal 2
   cd ui && RUTHERFORD_PORT=8081 npm run dev
   ```

   Set `ALLOWED_EMAILS=you@example.com` on the Go process to also exercise the
   allow-list.

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
