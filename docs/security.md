# Security controls

## Controls table

| Layer | Control | Where |
|---|---|---|
| Image | Multi-stage build, distroless, non-root, pinned base digest | `app/Dockerfile` |
| Image | Build-time scan, fails on HIGH/CRITICAL, SBOM | `.github/workflows/ci.yml` (`build-scan`, `sbom` jobs) |
| Image | Signing | not done - see "Known gaps" below |
| Supply chain | Actions pinned by commit SHA, not tag | `.github/workflows/*.yml` |
| Supply chain | Dependabot (gomod, github-actions, docker) | `.github/dependabot.yml` |
| Cluster network | Azure CNI Overlay + Cilium, default-deny ingress/egress, explicit allows for DNS and the app port | `infra/aks.tf`, `deploy/helm/ledger-api/templates/networkpolicy.yaml` |
| Cluster access | Local accounts enabled (see "Known gaps"), API server IP-restricted | `infra/aks.tf` (`api_server_access_profile`) |
| Pod | `restricted` Pod Security Admission on the `ledger` namespace | `deploy/namespace.yaml` |
| Pod | Non-root, read-only root filesystem, all capabilities dropped, no privilege escalation | `deploy/helm/ledger-api/templates/deployment.yaml` |
| Secrets | No password in the image, manifest, or CI. Secret lives in Key Vault, mounted via the CSI driver using the pod's workload identity | `infra/keyvault.tf`, `infra/identity.tf`, `deploy/helm/ledger-api/templates/secretproviderclass.yaml` |
| Admission | Azure Policy restricted-standards initiative (Audit) | enabled via `azure_policy_enabled = true` in `infra/aks.tf` |
| Admission | Kyverno: no `:latest`, requests/limits required (Enforce); signature verification (Audit, see gaps) | `deploy/policies/` |
| Vulnerability management | Nightly rescan of the deployed image, opens an issue on new findings | `.github/workflows/nightly-scan.yml` |
| Vulnerability management | In-cluster continuous scanning (Trivy Operator) | `deploy/observability/trivy-operator-values.yaml` |

## Known gaps (honest, not hidden)

- **Image signing (Phase 2b) was not implemented.** `ci.yml` builds, scans, and SBOMs the image, but never runs `cosign sign`. Because of this, Kyverno's `verify-image-signature` policy runs in `Audit`, not `Enforce` - enforcing it would reject the app's own unsigned image. In production this is a hard blocker before go-live, not an optional extra.
- **`local_account_disabled` is not set on the AKS cluster.** The project plan calls for disabling local Kubernetes accounts and requiring Entra ID + Azure RBAC for `kubectl` access (`kubelogin`). This was deliberately deferred in the session that built the rest of Phase 3-8: flipping it risks a lockout if Azure AD role propagation lags, and the risk/time trade-off wasn't worth it against a running bill. The role assignment needed (`Azure Kubernetes Service RBAC Cluster Admin`) and the `azure_active_directory_role_based_access_control` block are not yet in `aks.tf`.
- The Trivy nightly scan (`nightly-scan.yml`) targets the image on **GHCR** tagged by commit SHA, matching what `ci.yml`'s `push` job publishes. The image actually deployed to the lab's AKS cluster in this session was built directly into **ACR** with `az acr build` (a shortcut to deploy fast without first wiring the GHCR-to-ACR copy step from Phase 4). The two are not the same artifact. Closing this means either `cosign copy`-ing the GHCR image into ACR, or pointing the Helm chart at a GHCR-sourced digest once signing exists.

## Vulnerability triage process

1. Nightly scan or build-time scan finds a HIGH/CRITICAL, fixable (not `ignore-unfixed`) CVE.
2. An issue is opened automatically (nightly) or the PR is blocked (build-time).
3. Triage within 2 business days: is there a fixed version of the affected package/base image?
   - Yes: bump it, open a PR, let CI re-scan clean, merge.
   - No fix available yet: document the exception in this file (CVE ID, why it's accepted, re-check date), don't silently `ignore-unfixed` it away.
4. Re-deploy. Nightly scan confirms the finding is gone the next run.
