# Crossplane Provider for Ory Hydra — Specification

## Overview

A Crossplane v2 provider written in Go that manages Ory Hydra resources via the Hydra Admin API. The initial scope covers OAuth2 client lifecycle management (create, read, update, delete) with full field parity against the Hydra OAuth2Client model.

## Architecture Decisions

| Decision | Choice | Rationale |
|---|---|---|
| Crossplane version | v2 | Namespaced resources, MRDs, modern reconciler patterns |
| Go module path | `github.com/tjorri/crossplane-provider-hydra` | Personal GitHub namespace |
| API group | `hydra.crossplane.io` | Standard Crossplane community convention |
| API version | `v1alpha1` | Initial release, expect breaking changes |
| CRD kind | `OAuth2Client` | Clean, unambiguous within the Hydra provider scope |
| Resource scope | Namespaced | v2 default; enables RBAC isolation per team/namespace |
| Scaffolding | `crossplane/provider-template` + heavy pruning | Gets boilerplate for free; strip v1 leftovers and modernize |
| Hydra SDK | `github.com/ory/hydra-client-go` (official) | Full API coverage, types match OpenAPI spec |
| Hydra version | Latest stable | Track upstream releases |
| License | Apache 2.0 | Ecosystem standard for Crossplane providers |

## Managed Resource: OAuth2Client

### API Group & Version

```
apiVersion: hydra.crossplane.io/v1alpha1
kind: OAuth2Client
```

### Spec Fields (`forProvider`)

Full model parity with Hydra's OAuth2Client model (~50+ fields). Key fields include:

**Core:**
- `clientId` (optional) — If omitted, Hydra auto-generates a UUID4. Late-initialized back into the spec after creation.
- `clientName` — Human-readable name shown during authorization.
- `clientSecret` (optional, sensitive) — If provided by the user, used as-is. If omitted, Hydra auto-generates and the provider publishes it via connection secret.
- `clientUri` — URL of a web page with client info.

**OAuth2 Flow Configuration:**
- `grantTypes` — e.g., `["authorization_code", "refresh_token", "client_credentials"]`
- `responseTypes` — e.g., `["code", "token", "id_token"]`
- `redirectUris` — Must match exactly (no wildcards).
- `scope` — Space-delimited scope string.
- `audience` — Allowed audience values.
- `tokenEndpointAuthMethod` — `client_secret_basic`, `client_secret_post`, `private_key_jwt`, or `none`.

**Token Lifespans:**
- `authorizationCodeGrantAccessTokenLifespan`
- `authorizationCodeGrantIdTokenLifespan`
- `authorizationCodeGrantRefreshTokenLifespan`
- `clientCredentialsGrantAccessTokenLifespan`
- `implicitGrantAccessTokenLifespan`
- `implicitGrantIdTokenLifespan`
- `refreshTokenGrantAccessTokenLifespan`
- `refreshTokenGrantIdTokenLifespan`
- `refreshTokenGrantRefreshTokenLifespan`
- `deviceAuthorizationGrantAccessTokenLifespan`
- `jwtBearerGrantAccessTokenLifespan`

**OpenID Connect:**
- `backchannelLogoutUri`
- `backchannelLogoutSessionRequired`
- `frontchannelLogoutUri`
- `frontchannelLogoutSessionRequired`
- `requestObjectSigningAlgorithm`
- `userinfoSignedResponseAlg`
- `idTokenSignedResponseAlg`
- `subjectType` — `pairwise` or `public`
- `sectorIdentifierUri`
- `postLogoutRedirectUris`

**Advanced:**
- `contacts` — Contact emails.
- `logoUri`
- `policyUri`
- `tosUri`
- `corsOrigins` — Allowed CORS origins.
- `skipConsent` — Skip consent screen.
- `skipLogoutConsent` — Skip logout consent.
- `accessTokenStrategy` — `jwt` or `opaque`.
- `metadata` — Arbitrary JSON metadata.
- `jwks` — JSON Web Key Set (inline).
- `jwksUri` — URI to a JWKS endpoint.

### Status Fields (`atProvider`)

Server-set, read-only fields observed from Hydra:
- `clientId` — The resolved client ID (especially relevant when auto-generated).
- `createdAt`
- `updatedAt`
- `registrationAccessToken`
- `registrationClientUri`

### Drift Detection

- The provider reconciles all user-specified `forProvider` fields.
- Server-set fields (`createdAt`, `updatedAt`, `registrationAccessToken`, `registrationClientUri`) are **not** reconciled — they are observe-only and reflected in `atProvider`.
- If an external change is detected on a user-specified field, the provider issues an Update to restore the desired state.

### Client ID Lifecycle

1. **User provides `clientId`**: Used as-is during Create. Set as the external-name annotation.
2. **User omits `clientId`**: Hydra generates a UUID4. The provider late-initializes `forProvider.clientId` from the Hydra response and sets the external-name annotation.

### Connection Details (Published to Kubernetes Secret)

When `writeConnectionSecretToRef` is set, the provider publishes:

| Key | Source |
|---|---|
| `client_id` | The OAuth2 client ID |
| `client_secret` | The client secret (captured at creation or user-provided) |
| `token_endpoint` | Derived from the Hydra public URL |
| `authorization_endpoint` | Derived from the Hydra public URL |
| `issuer_url` | Derived from the Hydra public URL |

### Client Secret Handling

- If the user provides `clientSecret` in the spec, it is sent to Hydra during Create/Update.
- If omitted, Hydra auto-generates a secret. The provider captures it from the Create response and publishes it via the connection secret.
- The secret is only returned by Hydra on creation. If the connection secret is lost, the user must delete and recreate the OAuth2Client resource (or provide their own secret).

## ProviderConfig

```yaml
apiVersion: hydra.crossplane.io/v1alpha1
kind: ProviderConfig
metadata:
  name: default
spec:
  adminUrl: "http://hydra-admin.hydra.svc.cluster.local:4445"
  publicUrl: "http://hydra-public.hydra.svc.cluster.local:4444"
  credentials:
    source: None  # or "Secret"
    secretRef:
      namespace: crossplane-system
      name: hydra-bearer-token
      key: token
```

### Authentication Modes

1. **No auth (`source: None`)** — Hydra admin API is network-isolated (common in-cluster setup). Only `adminUrl` required.
2. **Bearer token (`source: Secret`)** — For deployments with an auth proxy in front of Hydra admin. Token loaded from a Kubernetes Secret.

### Fields

- `adminUrl` (required) — Hydra Admin API endpoint (port 4445).
- `publicUrl` (optional) — Hydra Public API endpoint (port 4444). Used to derive connection detail endpoints. If omitted, connection details won't include endpoint URLs.
- `credentials.source` — `None` or `Secret`.
- `credentials.secretRef` — Reference to a Secret containing a bearer token (when `source: Secret`).

### Multi-Instance Support

Each `OAuth2Client` resource references a `ProviderConfig` via `providerConfigRef`. Multiple ProviderConfigs can point to different Hydra instances (e.g., staging vs production).

## Project Structure

Starting from `crossplane/provider-template` with heavy pruning:

```
crossplane-provider-hydra/
├── apis/
│   ├── oauth2client/
│   │   └── v1alpha1/
│   │       ├── types.go              # OAuth2Client CRD types
│   │       ├── zz_generated.deepcopy.go
│   │       └── groupversion_info.go
│   └── v1alpha1/
│       ├── types.go                  # ProviderConfig types
│       ├── zz_generated.deepcopy.go
│       └── groupversion_info.go
├── internal/
│   ├── controller/
│   │   ├── oauth2client/
│   │   │   └── oauth2client.go       # ExternalConnecter + ExternalClient
│   │   └── config/
│   │       └── config.go             # ProviderConfig controller
│   ├── clients/
│   │   └── hydra.go                  # Hydra SDK wrapper
│   └── controller.go                 # Controller registration
├── cmd/
│   └── provider/
│       └── main.go                   # Provider entry point
├── e2e/
│   └── oauth2client_test.go          # End-to-end tests with testcontainers
├── package/
│   └── crossplane.yaml               # Crossplane package metadata
├── examples/
│   ├── oauth2client.yaml
│   └── providerconfig.yaml
├── .github/
│   └── workflows/
│       └── ci.yml                    # GitHub Actions: lint, test, build
├── go.mod
├── go.sum
├── Makefile
└── LICENSE                           # Apache 2.0
```

## Testing Strategy

### Approach: Red/Green TDD with testcontainers-go

Write failing tests first, then implement the provider to make them pass.

### Test Infrastructure

- **testcontainers-go** to spin up Ory Hydra in Docker.
- **SQLite in-memory** as the Hydra database backend (`DSN=memory`).
- Hydra runs with `--dev` flag for simplified setup (no TLS, relaxed security).
- No consent/login provider needed — tests only exercise the Admin API (client CRUD).
- Tests call the Hydra Admin API directly (port 4445) to verify state and simulate drift.

### Test Containers Setup

```go
// Pseudo-code for test setup
func setupHydra(ctx context.Context) (adminURL string, cleanup func()) {
    req := testcontainers.ContainerRequest{
        Image:        "oryd/hydra:<latest>",
        ExposedPorts: []string{"4445/tcp"},
        Cmd:          []string{"serve", "all", "--dev"},
        Env: map[string]string{
            "DSN":                            "memory",
            "URLS_SELF_ISSUER":               "http://localhost:4444",
            "URLS_LOGIN":                     "http://localhost:3000/login",
            "URLS_CONSENT":                   "http://localhost:3000/consent",
        },
        WaitingFor: wait.ForHTTP("/health/alive").WithPort("4445/tcp"),
    }
    // Start container, return mapped port URL
}
```

### Test Cases

#### CRUD Lifecycle
1. **Create**: Call `Create()` with a full OAuth2Client spec → verify client exists in Hydra via admin API.
2. **Observe (exists)**: Call `Observe()` → returns `ResourceExists: true`, `ResourceUpToDate: true`.
3. **Update**: Modify spec fields (e.g., change `scope`, add `redirectUris`) → call `Update()` → verify changes in Hydra.
4. **Observe (updated)**: Call `Observe()` after update → confirms `ResourceUpToDate: true` with new values.
5. **Delete**: Call `Delete()` → verify client no longer exists in Hydra.
6. **Observe (deleted)**: Call `Observe()` → returns `ResourceExists: false`.

#### Drift Detection
7. **External modification**: Create client via provider → modify client directly via Hydra admin API → call `Observe()` → returns `ResourceUpToDate: false`.
8. **Drift correction**: After detecting drift, call `Update()` → verify Hydra state matches spec again.
9. **Server-set fields ignored**: Modify `updatedAt` or `createdAt` in Hydra → call `Observe()` → still returns `ResourceUpToDate: true` (server-set fields are not reconciled).

#### Edge Cases
10. **Auto-generated client_id**: Create without `clientId` → verify `clientId` is late-initialized into the spec.
11. **User-provided client_secret**: Create with explicit `clientSecret` → verify Hydra uses it.
12. **Duplicate creation**: Call `Create()` for an already-existing client → verify idempotent behavior (no error, or appropriate error handling).
13. **Delete non-existent**: Call `Delete()` for a client that doesn't exist → verify no error (idempotent).
14. **Connection details**: Verify `client_id` and `client_secret` are returned from `Create()` for publishing.

## CI Pipeline (GitHub Actions)

```yaml
# .github/workflows/ci.yml
on: [push, pull_request]
jobs:
  lint:
    - golangci-lint
  test:
    - go test ./... (unit tests)
  e2e:
    - go test ./e2e/... (requires Docker)
  build:
    - go build ./cmd/provider/
```

## Dependencies

| Dependency | Purpose |
|---|---|
| `github.com/crossplane/crossplane-runtime` v2.x | Crossplane reconciler, interfaces, types |
| `github.com/ory/hydra-client-go` | Hydra Admin API client |
| `sigs.k8s.io/controller-runtime` | Kubernetes controller framework |
| `github.com/testcontainers/testcontainers-go` | Docker containers in tests |
| `k8s.io/apimachinery` | Kubernetes API types |

## Implementation Order (Red/Green)

1. **Scaffold project** — Initialize Go module, provider-template, prune.
2. **Define CRD types** — `OAuth2Client` and `ProviderConfig` types in Go.
3. **Write e2e test skeleton** — testcontainers setup, failing CRUD tests.
4. **Implement Hydra client wrapper** — Thin wrapper around the official SDK.
5. **Implement ExternalConnecter** — `Connect()` reads ProviderConfig, returns ExternalClient.
6. **Implement Observe()** — Make observe tests pass.
7. **Implement Create()** — Make create tests pass.
8. **Implement Update()** — Make update tests pass.
9. **Implement Delete()** — Make delete tests pass.
10. **Implement drift detection** — Make drift tests pass.
11. **Implement connection details** — Publish secrets from Create().
12. **Implement late initialization** — Auto-generated `clientId` flows.
13. **CI pipeline** — GitHub Actions workflow.
14. **Examples & documentation** — YAML examples for OAuth2Client and ProviderConfig.
