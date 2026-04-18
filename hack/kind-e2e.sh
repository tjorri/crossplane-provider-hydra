#!/usr/bin/env bash
# Full end-to-end test using Kind, Crossplane, and a real Hydra deployment.
# Works both locally and in CI.
#
# Usage:
#   ./hack/kind-e2e.sh          # run tests (creates cluster if needed)
#   ./hack/kind-e2e.sh teardown # delete the cluster

set -euo pipefail

CLUSTER_NAME="${KIND_E2E_CLUSTER:-provider-hydra-e2e}"
CONTROLLER_IMAGE="ghcr.io/tjorri/crossplane-provider-hydra-controller:e2e"
PACKAGE_IMAGE="crossplane-provider-hydra:e2e"
CROSSPLANE_VERSION="${CROSSPLANE_VERSION:-2.2.0}"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"

# --- helpers -----------------------------------------------------------------

log()  { echo "==> $*"; }
fail() { echo "FAIL: $*" >&2; exit 1; }

wait_for() {
  local desc="$1"; shift
  local timeout="${WAIT_TIMEOUT:-120}"
  log "Waiting for ${desc} (timeout ${timeout}s)..."
  for i in $(seq 1 "$timeout"); do
    if "$@" >/dev/null 2>&1; then
      log "${desc} — ready"
      return 0
    fi
    sleep 1
  done
  fail "${desc} — timed out after ${timeout}s"
}

check_tool() {
  command -v "$1" >/dev/null 2>&1 || fail "$1 is required but not found in PATH"
}

# --- teardown ----------------------------------------------------------------

if [[ "${1:-}" == "teardown" ]]; then
  log "Deleting Kind cluster ${CLUSTER_NAME}..."
  kind delete cluster --name "${CLUSTER_NAME}" 2>/dev/null || true
  exit 0
fi

# --- preflight ---------------------------------------------------------------

for tool in kind kubectl helm docker go controller-gen angryjet; do
  check_tool "$tool"
done

# --- cluster -----------------------------------------------------------------

if kind get clusters 2>/dev/null | grep -q "^${CLUSTER_NAME}$"; then
  log "Kind cluster ${CLUSTER_NAME} already exists, reusing"
else
  log "Creating Kind cluster ${CLUSTER_NAME}..."
  kind create cluster --name "${CLUSTER_NAME}" --wait 60s
fi

kubectl config use-context "kind-${CLUSTER_NAME}"

# --- install Crossplane ------------------------------------------------------

log "Installing Crossplane ${CROSSPLANE_VERSION}..."
helm repo add crossplane-stable https://charts.crossplane.io/stable 2>/dev/null || true
helm repo update crossplane-stable
helm upgrade --install crossplane crossplane-stable/crossplane \
  --namespace crossplane-system --create-namespace \
  --version "${CROSSPLANE_VERSION}" \
  --wait --timeout 120s

wait_for "Crossplane pods" \
  kubectl -n crossplane-system wait --for=condition=Ready pod -l app=crossplane --timeout=60s

# --- build provider ----------------------------------------------------------

cd "${ROOT_DIR}"

log "Generating CRDs and methods..."
make generate

log "Building controller image..."
docker build -t "${CONTROLLER_IMAGE}" .

log "Loading controller image into Kind..."
kind load docker-image "${CONTROLLER_IMAGE}" --name "${CLUSTER_NAME}"

# --- build and install xpkg --------------------------------------------------

check_tool crossplane || {
  log "Installing Crossplane CLI..."
  curl -sL "https://raw.githubusercontent.com/crossplane/crossplane/master/install.sh" | sh
  sudo mv crossplane /usr/local/bin/
}

log "Building xpkg..."
# Build xpkg in a temp directory to avoid dirtying the git tree.
XPKG_DIR=$(mktemp -d)
cp -r package/* "${XPKG_DIR}/"
sed "s|image:.*|image: ${CONTROLLER_IMAGE}|" "${XPKG_DIR}/crossplane.yaml" > "${XPKG_DIR}/crossplane.yaml.tmp"
mv "${XPKG_DIR}/crossplane.yaml.tmp" "${XPKG_DIR}/crossplane.yaml"

crossplane xpkg build \
  --package-root="${XPKG_DIR}" \
  --package-file=./provider-hydra-e2e.xpkg

rm -rf "${XPKG_DIR}"

log "Loading xpkg as OCI image into Kind..."
docker load -i ./provider-hydra-e2e.xpkg 2>/dev/null || true

# Install the provider by applying CRDs and deploying the controller directly.
# This validates the xpkg builds correctly while keeping the test fast.
log "Installing CRDs..."
kubectl apply -f package/crds/

log "Creating provider Deployment..."
kubectl create namespace provider-hydra 2>/dev/null || true
kubectl -n provider-hydra delete deployment provider-hydra 2>/dev/null || true

cat <<EOF | kubectl apply -f -
apiVersion: apps/v1
kind: Deployment
metadata:
  name: provider-hydra
  namespace: provider-hydra
spec:
  replicas: 1
  selector:
    matchLabels:
      app: provider-hydra
  template:
    metadata:
      labels:
        app: provider-hydra
    spec:
      serviceAccountName: provider-hydra
      containers:
        - name: provider
          image: ${CONTROLLER_IMAGE}
          imagePullPolicy: Never
          args: ["--debug"]
---
apiVersion: v1
kind: ServiceAccount
metadata:
  name: provider-hydra
  namespace: provider-hydra
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: provider-hydra
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: cluster-admin
subjects:
  - kind: ServiceAccount
    name: provider-hydra
    namespace: provider-hydra
EOF

wait_for "provider pod" \
  kubectl -n provider-hydra wait --for=condition=Ready pod -l app=provider-hydra --timeout=60s

# --- deploy Hydra ------------------------------------------------------------

log "Deploying Hydra..."
kubectl apply -f hack/manifests/hydra.yaml

wait_for "Hydra pod" \
  kubectl -n hydra wait --for=condition=Ready pod -l app=hydra --timeout=60s

# --- apply test resources ----------------------------------------------------

log "Applying test resources..."
kubectl apply -f hack/manifests/provider-test.yaml

# --- assertions --------------------------------------------------------------

log "Waiting for OAuth2Client to become ready..."
for i in $(seq 1 60); do
  synced=$(kubectl get oauth2client e2e-test-client -o jsonpath='{.status.conditions[?(@.type=="Synced")].status}' 2>/dev/null || echo "")
  ready=$(kubectl get oauth2client e2e-test-client -o jsonpath='{.status.conditions[?(@.type=="Ready")].status}' 2>/dev/null || echo "")
  if [[ "$synced" == "True" && "$ready" == "True" ]]; then
    break
  fi
  if [[ $i -eq 60 ]]; then
    log "OAuth2Client status:"
    kubectl get oauth2client e2e-test-client -o yaml 2>/dev/null || true
    log "Provider logs:"
    kubectl -n provider-hydra logs -l app=provider-hydra --tail=50 2>/dev/null || true
    fail "OAuth2Client did not become Synced+Ready within 60s"
  fi
  sleep 1
done
log "OAuth2Client is Synced and Ready"

# Verify the client exists in Hydra by port-forwarding and querying the admin API.
log "Verifying client exists in Hydra..."
kubectl -n hydra port-forward svc/hydra-admin 14445:4445 &
PF_PID=$!
sleep 2

HYDRA_RESPONSE=$(curl -sf "http://localhost:14445/admin/clients/e2e-test-client" 2>/dev/null || echo "")
kill "$PF_PID" 2>/dev/null || true
wait "$PF_PID" 2>/dev/null || true

if [[ -z "$HYDRA_RESPONSE" ]]; then
  fail "Client e2e-test-client not found in Hydra"
fi

CLIENT_NAME=$(echo "$HYDRA_RESPONSE" | python3 -c "import sys,json; print(json.load(sys.stdin).get('client_name',''))" 2>/dev/null || echo "")
if [[ "$CLIENT_NAME" != "E2E Test Client" ]]; then
  fail "Expected client_name 'E2E Test Client', got '${CLIENT_NAME}'"
fi
log "Client verified in Hydra: client_name=${CLIENT_NAME}"

# Verify connection secret was published.
log "Verifying connection secret..."
SECRET_DATA=$(kubectl get secret e2e-test-client-credentials -o jsonpath='{.data.client_id}' 2>/dev/null || echo "")
if [[ -z "$SECRET_DATA" ]]; then
  fail "Connection secret e2e-test-client-credentials not found or missing client_id"
fi
CLIENT_ID=$(echo "$SECRET_DATA" | base64 -d)
if [[ "$CLIENT_ID" != "e2e-test-client" ]]; then
  fail "Expected client_id 'e2e-test-client' in secret, got '${CLIENT_ID}'"
fi
log "Connection secret verified: client_id=${CLIENT_ID}"

# --- cleanup test resources (but keep cluster) --------------------------------

# Delete the OAuth2Client FIRST — it needs the ProviderConfig to connect to
# Hydra and delete the external resource via its finalizer.
log "Deleting OAuth2Client..."
kubectl delete oauth2client e2e-test-client --timeout=30s --ignore-not-found

wait_for "OAuth2Client deleted" \
  bash -c '! kubectl get oauth2client e2e-test-client 2>/dev/null'

log "Deleting ProviderConfig..."
kubectl delete providerconfig default --timeout=30s --ignore-not-found
rm -f ./provider-hydra-e2e.xpkg

# Verify the client was deleted from Hydra.
log "Verifying client was deleted from Hydra..."
kubectl -n hydra port-forward svc/hydra-admin 14445:4445 &
PF_PID=$!
sleep 2

DELETE_CHECK=$(curl -sf "http://localhost:14445/admin/clients/e2e-test-client" 2>/dev/null || echo "NOT_FOUND")
kill "$PF_PID" 2>/dev/null || true
wait "$PF_PID" 2>/dev/null || true

if [[ "$DELETE_CHECK" == "NOT_FOUND" ]] || echo "$DELETE_CHECK" | grep -q "Unable to locate"; then
  log "Client correctly deleted from Hydra after resource deletion"
else
  fail "Client still exists in Hydra after OAuth2Client deletion"
fi

echo ""
log "========================================="
log "  All Kind e2e tests passed!"
log "========================================="
echo ""
log "Cluster ${CLUSTER_NAME} is still running. To tear down:"
log "  ./hack/kind-e2e.sh teardown"
