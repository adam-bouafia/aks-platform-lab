# Step-by-step walkthrough, with screenshots

Every phase of this lab, the exact command run, and the screenshot proving the
result. Paths are relative to this file. Run these yourself against a fresh
`terraform apply` to reproduce any of it.

## Phase 2 - CI and supply chain

**Break-it drill:** a PR deliberately reintroducing an unhardened base image, to
prove the Trivy severity gate actually blocks a bad merge, not just logs a
warning.

```bash
git checkout -b drill/break-ci
# edit app/Dockerfile: swap the distroless final stage for a full, unpinned base image
git commit -am "drill: weaken Dockerfile to test CI gate"
git push origin drill/break-ci
gh pr create --title "drill: break-it test (do not merge)" --base main
```

![CI break-it drill: Trivy fails the build](screenshots/phase2-01-ci-break-it-drill-trivy-fail.png)
![CI pipeline summary: Build & Trivy Scan failed, downstream jobs skipped](screenshots/phase2-02-ci-pipeline-blocked-summary.png)
![PR merge blocked: required checks failing](screenshots/phase2-03-pr-merge-blocked.png)

## Phase 3 - Terraform Azure foundation

```bash
cd infra
terraform init
terraform plan -out=tfplan
terraform apply "tfplan"
```

![az aks show: cluster provisioningState Succeeded](screenshots/phase3-01-aks-show-succeeded.png)

```bash
terraform state list
```

![terraform state list, first apply (RG, VNet, subnet, AKS, ACR)](screenshots/phase3-02-terraform-state-list-initial.png)

```bash
kubectl get nodes
```

![kubectl get nodes, single-node cluster Ready](screenshots/phase3-03-kubectl-get-nodes-initial.png)

```bash
kubectl cluster-info
```

![kubectl cluster-info: control plane, CoreDNS, metrics-server all responding](screenshots/phase3-04-kubectl-cluster-info.png)

Azure Portal, the AKS-managed resource group (`MC_<rg>_<cluster>_<region>`):

![Azure Portal: the AKS-managed resource group (VMSS, load balancer, NSG, managed identity)](screenshots/phase3-05-azure-portal-resource-group.png)

After adding Key Vault, Log Analytics and workload identity (see
`infra/keyvault.tf`, `infra/identity.tf`, `infra/monitoring.tf`) and re-applying:

```bash
terraform state list
```

![terraform state list, full 16 resources (Key Vault, Log Analytics, workload identity, role assignments)](screenshots/phase3-06-terraform-state-list-final.png)

A single node couldn't fit AKS's own system pods alongside the app and
observability stack (see `notes/failures.md`) - scaled `node_count` to 2:

```bash
kubectl get nodes
```

![kubectl get nodes, scaled to 2 nodes](screenshots/phase3-07-kubectl-get-nodes-final.png)

## Phase 4 - Secure deployment

```bash
kubectl apply -f deploy/namespace.yaml
helm upgrade --install ledger-api ./deploy/helm/ledger-api \
  --set workloadIdentityClientId=$(terraform -chdir=infra output -raw ledger_api_identity_client_id) \
  --set keyVaultName=$(terraform -chdir=infra output -raw key_vault_name) \
  --set tenantId=$(terraform -chdir=infra output -raw tenant_id)
kubectl get pods -n ledger -o wide
```

![ledger-api pods Running, 1-node deployment](screenshots/phase4-01-pods-running-initial.png)

```bash
kubectl get networkpolicy -n ledger
```

![kubectl get networkpolicy -n ledger: default-deny-all, allow-dns-egress, allow-ledger-api-ingress](screenshots/phase4-02-networkpolicy.png)

```bash
kubectl get secretproviderclass -n ledger
```

![kubectl get secretproviderclass -n ledger: ledger-api-secrets](screenshots/phase4-03-secretproviderclass.png)

After the node-count fix, both replicas spread across both nodes
(`topologySpreadConstraints`):

```bash
kubectl get pods -n ledger -o wide
```

![ledger-api pods Running, spread across both nodes after the capacity fix](screenshots/phase4-04-pods-running-final-spread.png)

**No password anywhere.** The API key lives only in Key Vault; the pod gets it
via workload identity, never a Kubernetes Secret, CI variable, or the image:

```bash
kubectl port-forward -n ledger svc/ledger-api 8080:80 &
sleep 2
echo "--- without key ---"
curl -s -o /dev/null -w "%{http_code}\n" localhost:8080/api/v1/transactions
echo "--- with the real key from Key Vault ---"
API_KEY=$(az keyvault secret show --vault-name <kv-name> --name api-key --query value -o tsv)
curl -s -H "X-API-Key: $API_KEY" localhost:8080/api/v1/transactions
```

![curl: 401 without the key, 200 with the real key pulled from Key Vault](screenshots/phase4-05-curl-401-then-200-with-key.png)

**NetworkPolicy default-deny, verified live**, not just declared: a pod outside
the `ledger` namespace times out trying to reach the service.

```bash
kubectl run netpol-test --rm -i --restart=Never --image=busybox --timeout=15s -n default \
  -- wget -q -T 5 -O- ledger-api.ledger.svc.cluster.local
```

![NetworkPolicy default-deny: a pod outside ledger times out](screenshots/phase4-06-networkpolicy-default-deny-timeout.png)

**Break-it drill:** Pod Security Admission `restricted` rejects a pod with no
hardened `securityContext` before it ever starts.

```bash
kubectl run test-psa --image=nginx:1.27 -n ledger --restart=Never
```

![Pod Security Admission restricted: rejection message](screenshots/phase4-07-pod-security-admission-reject.png)

## Phase 5 - Guardrails

```bash
kubectl apply -f deploy/policies/
kubectl get namespacedvalidatingpolicy,namespacedimagevalidatingpolicy -n ledger
```

![Kyverno NamespacedValidatingPolicy and NamespacedImageValidatingPolicy, all Ready=true](screenshots/phase5-01-kyverno-policies-ready.png)

**Verified live, not just applied:** a `:latest`-tagged pod is rejected at
admission.

```bash
kubectl run test-latest --image=nginx:latest -n ledger --restart=Never \
  --overrides='{"spec":{"containers":[{"name":"test-latest","image":"nginx:latest",
  "securityContext":{"allowPrivilegeEscalation":false,"runAsNonRoot":true,"runAsUser":65532,
  "capabilities":{"drop":["ALL"]},"seccompProfile":{"type":"RuntimeDefault"}},
  "resources":{"requests":{"cpu":"10m","memory":"32Mi"},"limits":{"cpu":"50m","memory":"64Mi"}}}]}}'
```

![Kyverno rejects a :latest-tagged image](screenshots/phase5-02-kyverno-reject-latest-tag.png)

A pod with no resource requests/limits is rejected too:

```bash
kubectl run test-no-limits --image=nginx:1.27 -n ledger --restart=Never \
  --overrides='{"spec":{"containers":[{"name":"test-no-limits","image":"nginx:1.27",
  "securityContext":{"allowPrivilegeEscalation":false,"runAsNonRoot":true,"runAsUser":65532,
  "capabilities":{"drop":["ALL"]},"seccompProfile":{"type":"RuntimeDefault"}}}]}}'
```

![Kyverno rejects a pod with no resource requests/limits](screenshots/phase5-03-kyverno-reject-no-limits.png)

`verify-image-signature` runs in **Audit**, not Deny - Phase 2b (cosign signing)
was never wired into CI, so enforcing it would block the app's own unsigned
image. See `docs/decisions/ADR-003-policy-engines.md`.

## Phase 6 - Observability

```bash
helm upgrade --install kube-prometheus-stack prometheus-community/kube-prometheus-stack \
  -n monitoring --create-namespace -f deploy/observability/kube-prometheus-stack-values.yaml
kubectl apply -f deploy/observability/prometheusrule.yaml
kubectl get prometheusrule -n ledger
```

![kubectl get prometheusrule -n ledger: ledger-api-rules](screenshots/phase6-01-prometheusrule.png)

Import the dashboard, then open it in Grafana (port-forward
`kube-prometheus-stack-grafana` to a local port, log in `admin`/`admin`):

![Grafana: the ledger-api (RED) dashboard with live request rate, latency and ready-replica data](screenshots/phase6-02-grafana-ledger-api-red-dashboard.png)

The rest of `kube-prometheus-stack`'s bundled dashboards work too, proving the
whole stack - not just the one app - is observable:

![Grafana: bundled CoreDNS dashboard](screenshots/phase6-03-grafana-coredns-dashboard-a.png)
![Grafana: CoreDNS dashboard, cache hit rate](screenshots/phase6-04-grafana-coredns-dashboard-b.png)
![Grafana: Kubernetes / API server SLO dashboard](screenshots/phase6-05-grafana-apiserver-dashboard-a.png)
![Grafana: API server work queue and resource usage](screenshots/phase6-06-grafana-apiserver-dashboard-b.png)
![Grafana: Prometheus Overview, Target Sync legend explicitly showing serviceMonitor/ledger/ledger-api/0](screenshots/phase6-07-grafana-prometheus-overview-servicemonitor-target.png)
![Grafana: Prometheus Overview, storage and query stats](screenshots/phase6-08-grafana-prometheus-overview-storage.png)
![Grafana: Compute Resources / Multi-Cluster - CPU/memory commitment vs usage](screenshots/phase6-09-grafana-compute-resources-multicluster.png)
![Grafana: per-namespace CPU/memory breakdown - the evidence behind the single-node capacity failure in notes/failures.md](screenshots/phase6-10-grafana-compute-resources-per-namespace.png)
![Grafana: per-namespace network usage](screenshots/phase6-11-grafana-compute-resources-network.png)

## Phase 7 - Vulnerability management loop

```bash
helm upgrade --install trivy-operator aqua/trivy-operator \
  -n trivy-system --create-namespace -f deploy/observability/trivy-operator-values.yaml
kubectl get vulnerabilityreports -A
kubectl get configauditreports -n ledger
```

![kubectl get vulnerabilityreports -A: real reports across every AKS system workload](screenshots/phase7-01-vulnerabilityreports-all.png)
![kubectl get configauditreports -n ledger: NetworkPolicies, the ReplicaSet and the Service all audited](screenshots/phase7-02-configauditreports-ledger.png)

```bash
kubectl get vulnerabilityreports -A -o custom-columns='NAMESPACE:.metadata.namespace,NAME:.metadata.name,CRITICAL:.report.summary.criticalCount,HIGH:.report.summary.highCount,MEDIUM:.report.summary.mediumCount,LOW:.report.summary.lowCount'
```

![kubectl get vulnerabilityreports with a CVE count table: real CRITICAL/HIGH/MEDIUM/LOW findings, including 2 CRITICAL + 9 HIGH on an AKS system daemonset](screenshots/phase7-03-vulnerabilityreports-cve-table.png)

Trivy Operator's `ledger-api` image scan itself hit an ACR-authentication gap
(same root cause as Kyverno's signature check above) - documented, not hidden,
in `docs/security.md`'s "Known gaps" section.

## Tear down

```bash
cd infra
terraform plan -destroy -out=tfdestroy
terraform apply "tfdestroy"
az resource list -g rg-ledger-lab -o table   # confirm empty
```
