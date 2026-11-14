# 5-minute interview demo

1. **The image** (30s): `docker images` naive vs. hardened - size and CVE count
   from the README table. "Took it from ~1 GB with hundreds of CVEs to a
   distroless, non-root, no-shell image."

2. **The infra** (45s): `terraform state list` in `infra/` - everything from
   the resource group to the federated identity credential is code. Point at
   `infra/aks.tf`'s `network_profile` block: "Azure CNI Overlay with Cilium -
   pods never touch the VNet's address space, and NetworkPolicy is enforced
   natively."

3. **The secret, no password anywhere** (60s):
   ```bash
   kubectl port-forward -n ledger svc/ledger-api 8080:80 &
   curl -s -o /dev/null -w "%{http_code}\n" localhost:8080/api/v1/transactions   # 401, no key
   API_KEY=$(az keyvault secret show --vault-name <kv-name> --name api-key --query value -o tsv)
   curl -H "X-API-Key: $API_KEY" localhost:8080/api/v1/transactions              # 200, real data
   ```
   "That key never touches a Kubernetes Secret, a CI variable, or the image.
   The pod's workload identity exchanges a Kubernetes token for an Entra
   token, and the Secrets Store CSI driver pulls it straight from Key Vault."

4. **A break-it drill** (60s): show the Pod Security Admission rejection
   message from `notes/failures.md` (trying to run a pod with
   `allowPrivilegeEscalation` unset in the `ledger` namespace). "Admission
   rejects it before it ever starts, not after."

5. **NetworkPolicy proof** (45s): a pod outside the `ledger` namespace times
   out hitting the service; a pod inside, properly labelled for the
   restricted PSA level, works. "Default-deny, explicit allows only."

6. **Observability** (30s): Grafana RED dashboard
   (`deploy/observability/ledger-api-dashboard.json`) with live request rate,
   error ratio and latency; mention the `PrometheusRule` alert that fires at
   a 5% error ratio.

7. **Vulnerability loop** (30s): `nightly-scan.yml` and the Trivy Operator -
   "a clean scan at build time isn't enough, CVEs get published after you
   ship, so this rescans the deployed image every night and opens an issue."
