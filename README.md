# crossplane-provider-hydra

A [Crossplane](https://crossplane.io/) v2 provider for managing [Ory Hydra](https://www.ory.sh/hydra/) OAuth2 clients as Kubernetes resources.

## Overview

This provider enables declarative management of Ory Hydra OAuth2 clients through Kubernetes CRDs. It supports:

- Full CRUD lifecycle for OAuth2 clients
- Drift detection and automatic reconciliation
- Auto-generated or user-provided client IDs and secrets
- Connection secret publishing (client_id, client_secret, endpoints)
- Multiple Hydra instances via ProviderConfig

## Installation

### Prerequisites

- A Kubernetes cluster with [Crossplane](https://docs.crossplane.io/v2.2/getting-started/install-crossplane/) v2 installed
- An [Ory Hydra](https://www.ory.sh/hydra/) instance reachable from the cluster

### Install the provider

```yaml
apiVersion: pkg.crossplane.io/v1
kind: Provider
metadata:
  name: provider-hydra
spec:
  package: ghcr.io/tjorri/crossplane-provider-hydra:latest
```

```bash
kubectl apply -f - <<EOF
apiVersion: pkg.crossplane.io/v1
kind: Provider
metadata:
  name: provider-hydra
spec:
  package: ghcr.io/tjorri/crossplane-provider-hydra:latest
EOF
```

Wait for the provider to become healthy:

```bash
kubectl get providers.pkg.crossplane.io provider-hydra
```

### Install from source

To build and load the provider into a local cluster (e.g. Kind):

```bash
# Build the controller image
make docker-build VERSION=local
kind load docker-image ghcr.io/tjorri/crossplane-provider-hydra-controller:local

# Build the xpkg and load it
make xpkg-build VERSION=local
crossplane xpkg push crossplane-provider-hydra:local -f crossplane-provider-hydra-local.xpkg
```

Then install with the local image:

```yaml
apiVersion: pkg.crossplane.io/v1
kind: Provider
metadata:
  name: provider-hydra
spec:
  package: crossplane-provider-hydra:local
```

### Install CRDs only (standalone / development)

If you prefer to run the provider outside of Crossplane's package manager:

```bash
# Generate and install CRDs
make generate
kubectl apply -f package/crds/

# Run the provider locally against your cluster
go run ./cmd/provider/ --debug
```

## Quickstart

### 1. Configure the provider

```yaml
apiVersion: hydra.crossplane.io/v1alpha1
kind: ProviderConfig
metadata:
  name: default
spec:
  adminUrl: "http://hydra-admin.hydra.svc.cluster.local:4445"
  publicUrl: "http://hydra-public.hydra.svc.cluster.local:4444"
  credentials:
    source: None
```

### 2. Create an OAuth2 client

```yaml
apiVersion: hydra.crossplane.io/v1alpha1
kind: OAuth2Client
metadata:
  name: my-app
  namespace: default
spec:
  providerConfigRef:
    name: default
  writeConnectionSecretToRef:
    name: my-app-oauth2-credentials
    namespace: default
  forProvider:
    clientId: "my-app"
    clientName: "My Application"
    grantTypes:
      - authorization_code
      - refresh_token
    responseTypes:
      - code
    redirectUris:
      - "https://myapp.example.com/callback"
    scope: "openid profile email"
    tokenEndpointAuthMethod: "client_secret_basic"
```

The provider will create the client in Hydra and publish credentials to a Kubernetes Secret:

```bash
kubectl get secret my-app-oauth2-credentials -o jsonpath='{.data.client_secret}' | base64 -d
```

## ProviderConfig

### Authentication modes

**No auth** (Hydra admin API is network-isolated):

```yaml
spec:
  adminUrl: "http://hydra-admin:4445"
  credentials:
    source: None
```

**Bearer token** (auth proxy in front of Hydra admin):

```yaml
spec:
  adminUrl: "https://hydra-admin.example.com"
  credentials:
    source: Secret
    secretRef:
      namespace: crossplane-system
      name: hydra-bearer-token
      key: token
```

### Multiple Hydra instances

Each OAuth2Client references a ProviderConfig via `providerConfigRef`, so you can target different Hydra instances:

```yaml
# staging
apiVersion: hydra.crossplane.io/v1alpha1
kind: ProviderConfig
metadata:
  name: staging
spec:
  adminUrl: "http://hydra-admin.staging:4445"
---
# production
apiVersion: hydra.crossplane.io/v1alpha1
kind: ProviderConfig
metadata:
  name: production
spec:
  adminUrl: "http://hydra-admin.production:4445"
```

## OAuth2Client spec reference

The `forProvider` spec has full parity with the Hydra OAuth2Client model. Key fields:

| Field | Type | Description |
|---|---|---|
| `clientId` | string | OAuth2 client ID (auto-generated if omitted) |
| `clientName` | string | Human-readable name |
| `clientSecret` | string | Client secret (auto-generated if omitted) |
| `grantTypes` | []string | Allowed grant types |
| `responseTypes` | []string | Allowed response types |
| `redirectUris` | []string | Allowed redirect URIs |
| `scope` | string | Space-delimited scopes |
| `audience` | []string | Allowed audiences |
| `tokenEndpointAuthMethod` | string | Auth method: `client_secret_basic`, `client_secret_post`, `private_key_jwt`, `none` |
| `accessTokenStrategy` | string | `jwt` or `opaque` |
| `skipConsent` | bool | Skip consent screen |
| `subjectType` | string | `pairwise` or `public` |

See [`apis/oauth2client/v1alpha1/types.go`](apis/oauth2client/v1alpha1/types.go) for the complete field list including token lifespans, OIDC settings, CORS origins, and more.

## Connection details

When `writeConnectionSecretToRef` is set, the provider publishes:

| Key | Description |
|---|---|
| `client_id` | The OAuth2 client ID |
| `client_secret` | The client secret (captured at creation) |
| `token_endpoint` | Token endpoint URL (if `publicUrl` is set) |
| `authorization_endpoint` | Authorization endpoint URL |
| `issuer_url` | Issuer URL |

## Development

### Prerequisites

- Go 1.25+
- Docker (for e2e tests)
- [controller-gen](https://book.kubebuilder.io/reference/controller-gen)
- [angryjet](https://github.com/crossplane/crossplane-tools)

### Build

```bash
make build
```

### Run tests

```bash
# Unit tests (fast, no Docker required)
make test

# E2E tests (spins up Hydra in Docker via testcontainers)
make test-e2e
```

### Regenerate code

After modifying CRD types in `apis/`:

```bash
make generate
```

### Releasing

This project uses [semantic versioning](https://semver.org/) with [release-please](https://github.com/googleapis/release-please) for automated version management.

**How it works:**

1. Merge PRs with [Conventional Commits](https://www.conventionalcommits.org/) (`feat:`, `fix:`, `feat!:` for breaking changes)
2. Release-please automatically opens a Release PR that bumps the version and updates `CHANGELOG.md`
3. Merge the Release PR — release-please creates the git tag and GitHub Release
4. The tag triggers the publish workflow, which builds and pushes all artifacts

**Manual release** (if needed):

```bash
git tag v0.1.0
git push origin v0.1.0
```

The publish workflow will:

1. Run lint and unit tests
2. Generate CRDs
3. Build and push the controller image to `ghcr.io/tjorri/crossplane-provider-hydra-controller:<tag>`
4. Build the Crossplane package (xpkg)
5. Push the xpkg to `ghcr.io/tjorri/crossplane-provider-hydra:<tag>`
6. Attach the xpkg to the GitHub Release

**Setup:** Release-please requires a `RELEASE_TOKEN` repository secret containing a Personal Access Token with `contents:write` and `pull-requests:write` scopes.

### Image layout

| Image | Purpose |
|---|---|
| `ghcr.io/tjorri/crossplane-provider-hydra:<tag>` | Crossplane package (xpkg) — this is what users reference in `spec.package` |
| `ghcr.io/tjorri/crossplane-provider-hydra-controller:<tag>` | Controller binary — pulled automatically by Crossplane |

## License

Apache 2.0 - see [LICENSE](LICENSE).
