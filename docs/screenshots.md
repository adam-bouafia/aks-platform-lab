# Screenshots

Evidence for each phase, in order. Paths are relative to this file.

## Phase 2 - CI and supply chain

**Break-it drill:** a PR deliberately reintroducing an unhardened base image, blocked by the Trivy severity gate.

![CI break-it drill: Trivy fails the build](screenshots/phase2-01-ci-break-it-drill-trivy-fail.png)
![CI pipeline summary: Build & Trivy Scan failed, downstream jobs skipped](screenshots/phase2-02-ci-pipeline-blocked-summary.png)
![PR merge blocked: required checks failing](screenshots/phase2-03-pr-merge-blocked.png)

## Phase 3 - Terraform Azure foundation

![az aks show: cluster provisioningState Succeeded](screenshots/phase3-01-aks-show-succeeded.png)
![terraform state list, first apply (RG, VNet, subnet, AKS, ACR)](screenshots/phase3-02-terraform-state-list-initial.png)
![kubectl get nodes, single-node cluster Ready](screenshots/phase3-03-kubectl-get-nodes-initial.png)
![kubectl cluster-info: control plane, CoreDNS, metrics-server all responding](screenshots/phase3-04-kubectl-cluster-info.png)
![Azure Portal: the AKS-managed resource group (VMSS, load balancer, NSG, managed identity)](screenshots/phase3-05-azure-portal-resource-group.png)
![terraform state list, full 16 resources (Key Vault, Log Analytics, workload identity, role assignments)](screenshots/phase3-06-terraform-state-list-final.png)
![kubectl get nodes, scaled to 2 nodes](screenshots/phase3-07-kubectl-get-nodes-final.png)

## Phase 4 - Secure deployment

![ledger-api pods Running, 1-node deployment](screenshots/phase4-01-pods-running-initial.png)
![kubectl get networkpolicy -n ledger: default-deny-all, allow-dns-egress, allow-ledger-api-ingress](screenshots/phase4-02-networkpolicy.png)
![kubectl get secretproviderclass -n ledger: ledger-api-secrets](screenshots/phase4-03-secretproviderclass.png)
![ledger-api pods Running, spread across both nodes after the capacity fix](screenshots/phase4-04-pods-running-final-spread.png)

## Phase 5 - Guardrails

![Kyverno NamespacedValidatingPolicy and NamespacedImageValidatingPolicy, all Ready=true](screenshots/phase5-01-kyverno-policies-ready.png)

## Phase 6 - Observability

![kubectl get prometheusrule -n ledger: ledger-api-rules](screenshots/phase6-01-prometheusrule.png)
![Grafana: the ledger-api (RED) dashboard with live request rate, latency and ready-replica data](screenshots/phase6-02-grafana-ledger-api-red-dashboard.png)
![Grafana: bundled CoreDNS dashboard, proving the whole stack scrapes cluster DNS too](screenshots/phase6-03-grafana-coredns-dashboard-a.png)
![Grafana: CoreDNS dashboard, cache hit rate](screenshots/phase6-04-grafana-coredns-dashboard-b.png)
![Grafana: Kubernetes / API server SLO dashboard](screenshots/phase6-05-grafana-apiserver-dashboard-a.png)
![Grafana: API server work queue and resource usage](screenshots/phase6-06-grafana-apiserver-dashboard-b.png)
![Grafana: Prometheus Overview, Target Sync legend explicitly showing serviceMonitor/ledger/ledger-api/0](screenshots/phase6-07-grafana-prometheus-overview-servicemonitor-target.png)
![Grafana: Prometheus Overview, storage and query stats](screenshots/phase6-08-grafana-prometheus-overview-storage.png)
![Grafana: Compute Resources / Multi-Cluster - CPU/memory commitment vs usage](screenshots/phase6-09-grafana-compute-resources-multicluster.png)
![Grafana: per-namespace CPU/memory breakdown (gatekeeper-system, kube-system, kyverno, ledger, monitoring, trivy-system) - the evidence behind the single-node capacity failure in notes/failures.md](screenshots/phase6-10-grafana-compute-resources-per-namespace.png)
![Grafana: per-namespace network usage](screenshots/phase6-11-grafana-compute-resources-network.png)

## Phase 7 - Vulnerability management loop

![kubectl get vulnerabilityreports -A: real reports across every AKS system workload](screenshots/phase7-01-vulnerabilityreports-all.png)
![kubectl get configauditreports -n ledger: NetworkPolicies, the ReplicaSet and the Service all audited](screenshots/phase7-02-configauditreports-ledger.png)
![kubectl get vulnerabilityreports -A with a CVE count table: real CRITICAL/HIGH/MEDIUM/LOW findings, including 2 CRITICAL + 9 HIGH on an AKS system daemonset](screenshots/phase7-03-vulnerabilityreports-cve-table.png)

## Known gaps not covered by a screenshot

These were verified directly in the terminal during the session but not separately
captured as images (the cluster was torn down right after to stop billing):

- The `curl` sequence proving no password anywhere: `401` without the API key,
  `200` with the real key pulled live from Key Vault.
- The NetworkPolicy default-deny proof: a pod outside `ledger` timing out against
  the service.
- The Kyverno `kubectl apply` rejection messages for a `:latest`-tagged pod and a
  pod with no resource requests/limits.
- The Pod Security Admission `restricted` rejection message (break-it drill).

All of the above are quoted verbatim, with full command and output, in this
session's transcript and in `notes/failures.md`. They're straightforward to
re-capture on a future `terraform apply` if screenshots are needed later.
