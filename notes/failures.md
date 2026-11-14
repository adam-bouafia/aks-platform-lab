# Failures log

Format: Symptom / Looked at / Root cause / Fix / Prevent

## 2026-11-13 Docker daemon not running

Symptom:     `docker build`, `docker images`, `docker run` and `trivy image` all failed
             with "Cannot connect to the Docker daemon at unix:///var/run/docker.sock.
             Is the docker daemon running?" (plus noisy fallback errors from trivy
             trying containerd, podman and Docker Hub).
Looked at:   `systemctl is-active docker` -> inactive; `systemctl is-enabled docker` ->
             disabled; `groups $USER` -> already includes `docker`, so permissions were
             not the problem.
Root cause:  The docker service was installed but never enabled or started on this
             machine.
Fix:         `sudo systemctl enable --now docker`
Prevent:     When any docker/trivy command fails with a socket-connection error, check
             `systemctl is-active docker` first, before assuming it's a build or image
             problem.

## 2026-11-13 hadolint "file does not exist"

Symptom:     `hadolint app/Dockerfile.naive` and `hadolint app/Dockerfile` both failed
             with `withBinaryFile: does not exist (No such file or directory)`.
Looked at:   `pwd` showed the shell was already inside `app/`, and `ls Dockerfile*`
             confirmed the files are at `app/Dockerfile` and `app/Dockerfile.naive`,
             not `app/app/Dockerfile*`.
Root cause:  Path doubled up: ran the command with an `app/` prefix while already
             inside the `app/` directory, so hadolint looked for a path that does
             not exist.
Fix:         Run hadolint with the path relative to the current directory
             (`hadolint Dockerfile.naive`, `hadolint Dockerfile`) instead of
             repeating the `app/` prefix.
Prevent:     Check `pwd` before pasting a path from higher up the repo tree into a
             command run from inside a subdirectory.

## 2026-11-13 CI failed: unable to resolve pinned action SHA

Symptom:     GitHub Actions run failed immediately in "Prepare all required actions"
             with: Error: Unable to resolve action
             `terraform-linters/setup-tflint@99297851292694f0418c5f08843f29a853c529a1`,
             unable to find version `99297851292694f0418c5f08843f29a853c529a1`.
Looked at:   `git ls-remote --tags https://github.com/terraform-linters/setup-tflint.git`
             to list the real commit SHA behind the v4.1.1 tag; it did not match what
             was in ci.yml. Checked all 12 pinned actions in the file the same way
             (for annotated tags, the real commit is the `^{}` peeled ref, not the
             tag object itself) and found 6 wrong: setup-tflint, trivy-action,
             codeql-action/upload-sarif, download-artifact, sbom-action, login-action.
Root cause:  Several commit SHAs pinned in the workflow did not correspond to the
             release tag named in their comment - likely fabricated/guessed rather
             than copied from the real repo (one, docker/login-action, was off by
             only a few characters from the real SHA, easy to miss on a visual read).
Fix:         Looked up the real commit for each tag with
             `git ls-remote --tags https://github.com/<owner>/<repo>.git` and replaced
             every wrong SHA with the verified one.
Prevent:     Never trust a generated `@<sha> # <tag>` pin at face value. Verify every
             pin with `git ls-remote --tags` (using the `^{}` peeled SHA for annotated
             tags) before committing a workflow file, especially for less common
             actions.

## 2026-11-13 CI failed: trivy-action's own pinned version was broken upstream

Symptom:     After fixing the setup-tflint SHA, CI failed again in a different step:
             Error: Unable to resolve action `aquasecurity/setup-trivy@v0.2.1`, unable
             to find version `v0.2.1`. That action is not referenced anywhere in
             ci.yml directly.
Looked at:   Fetched aquasecurity/trivy-action's own action.yaml at the pinned commit
             (v0.28.0) and found it internally calls `uses: aquasecurity/setup-trivy@v0.2.1`
             as a nested step. `git ls-remote --tags` on aquasecurity/setup-trivy showed
             the oldest tag still present is v0.2.6 - v0.2.1 had been deleted/retagged
             upstream. Checked the latest trivy-action release (v0.36.0) and confirmed
             its action.yaml now pins setup-trivy to a commit SHA for v0.2.6, which
             does exist.
Root cause:  Our SHA pin for trivy-action@0.28.0 was valid, but that specific old
             release of trivy-action itself has a broken internal dependency (an
             upstream tag it references no longer exists). Pinning by SHA protects
             against the tag moving, but not against the pinned release depending on
             something that later disappears.
Fix:         Upgraded the trivy-action pin to v0.36.0 (SHA a9c7b0f06e461e9d4b4d1711f154ee024b8d7ab8),
             whose own internal setup-trivy reference is healthy.
Prevent:     When a pinned action fails to resolve a *different* action than the one
             named in the failing step, suspect a nested/composite action dependency.
             Check the failing action's own action.yaml/action.yml at the pinned SHA.
             Prefer reasonably recent releases of third-party actions, since very old
             pinned versions can depend on upstream refs that get cleaned up over time.

## 2026-11-13 CI failed: sbom-action's "path" input double-prefixed the scan target

Symptom:     Generate CycloneDX SBOM job failed with "Error: The process
             '/opt/hostedtoolcache/syft/1.19.0/x64/syft' failed with exit code 1".
             The printed command showed the real problem:
             `syft scan dir:docker-archive:/tmp/image.tar -o cyclonedx-json`
             - two scheme prefixes glued together, which Syft can't parse.
Looked at:   The workflow passed `path: docker-archive:/tmp/image.tar` to
             anchore/sbom-action. Checked the action's README: `path` is documented
             as "a path on the filesystem to scan", separate from `image` ("a
             container image to scan"). Downloaded the action's compiled
             dist/runSyftAction/index.js at the pinned SHA and found the exact
             branching: the `image` input is passed to syft unprefixed, `file` gets
             `file:` prepended, and `path` always gets `dir:` prepended - regardless
             of what string you put in it.
Root cause:  Used the wrong input. `path` is meant for a plain directory and always
             gets `dir:` glued on the front, so giving it an already-complete
             `docker-archive:...` scheme produced an invalid double-prefixed target.
Fix:         Switched the step to use `image: docker-archive:/tmp/image.tar` instead
             of `path:` - that input is forwarded to syft as-is, which is what a
             scheme-prefixed scan target needs.
Prevent:     When a wrapper action's documented inputs don't make the exact command
             it will run obvious, read its actual source (the compiled dist/*.js for
             a Node action) rather than guessing from the README's short description
             alone - the parameter names can look interchangeable but build very
             different commands underneath.

## 2026-11-14 Terraform apply failed: region not allowed for student subscription

Symptom:     `terraform apply` created the resource group in `westeurope` fine,
             then failed creating the VNet with 403 Forbidden:
             "RequestDisallowedByAzure: Resource ... was disallowed by Azure:
             This policy maintains a set of best available regions where your
             subscription can deploy resources."
Looked at:   `az policy assignment list` on the subscription, found a system
             policy assignment `sys.regionrestriction` ("Allowed resource
             deployment regions") with `listOfAllowedLocations`:
             francecentral, spaincentral, swedencentral, germanywestcentral,
             italynorth. `westeurope` is not in that list.
Root cause:  "Azure for Students" subscriptions get a built-in Azure policy
             restricting which regions resources can actually be deployed into
             - a quota/abuse-prevention measure, not a bug in the Terraform
             code. Resource-group creation itself succeeded in westeurope
             because RG location metadata isn't covered by this policy; any
             regional resource (VNet, AKS, etc.) is.
Fix:         Changed `location` to `germanywestcentral` (closest allowed region
             to NL) on the resource group; everything else already referenced
             it via `azurerm_resource_group.main.location` instead of a
             hardcoded string, so only one line needed to change. Resource
             group had to be destroyed and recreated (location forces
             replacement), then VNet + subnet created cleanly.
Prevent:     Before deploying any new Azure resource type in this project
             (ACR, Key Vault, Log Analytics, AKS), check it against this same
             allowed-region list first - don't assume westeurope works just
             because the resource group accepted it.

## 2026-11-14 Terraform apply failed: provider version drift (field renames + new required block)

Symptom:     `terraform plan` on `aks.tf` failed with two errors:
             `Insufficient node_provisioning_profile blocks` ("At least 1
             node_provisioning_profile blocks are required") and
             `Unsupported argument` on `network_dataplane` ("Did you mean
             network_data_plane?").
Looked at:   `terraform providers schema -json` on `azurerm_kubernetes_cluster`
             confirmed `node_provisioning_profile` has `min_items: 1` in
             provider v5.14.0, with fields `mode` and `default_node_pools`.
Root cause:  Written against older/tutorial field names and an older provider
             schema. azurerm v5.x renamed `network_dataplane` to
             `network_data_plane`, and added a newer AKS feature (Node
             Autoprovisioning) that the schema now requires an explicit
             opt-in/opt-out for via `node_provisioning_profile` - no implicit
             default anymore.
Fix:         Renamed the field to `network_data_plane`, added
             `node_provisioning_profile { mode = "Manual" }` (explicit opt-out
             of AKS auto-provisioning, since nodes are managed via
             `default_node_pool` instead).
Prevent:     When a resource block errors with "Unsupported argument" or a
             "required block missing" right after bumping/installing a new
             provider version, check `terraform providers schema -json` for
             the resource in question rather than assuming the Terraform code
             itself is wrong - provider schemas change between major/minor
             releases.

## 2026-11-14 Terraform apply failed: Microsoft.ContainerService provider not registered

Symptom:     `terraform apply` on `aks.tf` failed creating the cluster with
             409 Conflict: "MissingSubscriptionRegistration: The subscription
             is not registered to use namespace 'Microsoft.ContainerService'."
Looked at:   This is a fresh "Azure for Students" subscription that had never
             created an AKS cluster (or any ContainerService resource) before.
Root cause:  Azure resource providers (namespaces like Microsoft.ContainerService,
             Microsoft.Network, etc.) must be explicitly registered per
             subscription before their resource types can be created. Brand
             new or rarely-used subscriptions often have common ones
             registered already (Network, Storage) but less common ones
             (ContainerService) not yet.
Fix:         `az provider register --namespace Microsoft.ContainerService`,
             then polled `az provider show -n Microsoft.ContainerService
             --query registrationState` until it reported `Registered`
             (took about a minute). Re-ran `terraform apply` with no code
             changes - it proceeded past the point of the earlier error.
Prevent:     On a fresh subscription, before the first `terraform apply` that
             uses a resource type you haven't deployed before on it, check
             `az provider list --query "[?registrationState!='Registered']"`
             for anything relevant and register it ahead of time instead of
             discovering it mid-apply.

## 2026-11-14 AKS create stuck 45+ min, then quota-blocked on every allowed SKU family

Symptom:     First `terraform apply` for the AKS cluster sat at
             "CreatingAgentPools: 0/1 nodes completed" (checked via
             `az aks operation show`) for 45+ minutes with zero progress -
             abnormal, normal node provisioning reaches a node within
             5-10 minutes. Destroyed and recreated clean
             (`terraform destroy` then `apply`); the second attempt failed
             fast and explicitly: "ErrCode_InsufficientVCPUQuota ... 0
             remaining for family StandardDasv7Family". Switching
             `vm_size` to `Standard_D2s_v3` then failed differently:
             "The VM size of Standard_D2s_v3 is not allowed in your
             subscription in location 'germanywestcentral'" with a long
             allow-list of permitted SKUs (all v6/v7-generation D/E/F
             families, DC v5/v6, L v4/v5, M-series, and GPU NC/NV only -
             no v3/v4 generation anywhere in it).
Looked at:   `az vm list-usage --location <region>` for all 5 of the
             region-policy-allowed regions (germanywestcentral,
             francecentral, spaincentral, swedencentral, italynorth),
             cross-checked against the AKS error's own allowed-SKU list.
             Every non-GPU family AKS is permitted to use
             (DASv7/DSv7/Dv7/ESv7/etc.) shows limit=0, current=0 in every
             one of the 5 regions. Only GPU families (NC/NV) have nonzero
             quota (6), which isn't appropriate for this lab (cost, and
             GPU quota frequently needs separate manual approval even when
             the number shown is nonzero).
Root cause:  Two separate, compounding constraints on an "Azure for
             Students" subscription: (1) a region-restriction policy
             limiting deployment to 5 regions (see the earlier region
             failure entry), and (2) AKS only permits newer-generation
             (v6/v7+) VM SKUs for node pools on this subscription, but
             this subscription was never granted any vCPU quota for those
             newer-generation families in any of the 5 allowed regions -
             only older-generation families (which AKS won't use at all)
             and GPU families have quota. The 45-minute hang on the first
             attempt was most likely the same underlying quota exhaustion
             manifesting as a stall before Azure's backend gave up and
             returned the explicit 400, rather than a separate bug.
Fix:         Not fixed same night - this requires an actual Azure quota
             increase request (Portal -> Quotas, or Help+support -> New
             support request -> Service and subscription limits), not a
             Terraform or naming change. Reverted `vm_size` back to
             `Standard_D2as_v7` (confirmed on AKS's allow-list, just
             needs quota) rather than leaving it on `Standard_D2s_v3`
             (not on the allow-list at all, a dead end regardless of
             quota).
Prevent:     On a free/student/trial subscription, before relying on any
             specific VM family for a managed service like AKS, check
             both (a) the service's own allowed-SKU list for the
             subscription (triggering the error once is the fastest way
             to get it) and (b) `az vm list-usage` for that exact family
             in that exact region - a "create" that hangs for an unusually
             long time with no progress is itself a signal to check quota
             before assuming it's just slow.

## 2026-11-13 Break-it drill: unhardened image rejected by CI

Symptom:     N/A - deliberate test. Opened a PR replacing the distroless final
             stage with `FROM golang:1.26` to reintroduce ~183 HIGH/CRITICAL CVEs.
Looked at:   Build & Trivy Scan job on the PR; failed at the HIGH/CRITICAL gate step.
Root cause:  N/A - intentional regression to verify the CI gate actually blocks
             a bad image rather than just passing everything.
Fix:         Closed the PR without merging; branch deleted.
Prevent:     This IS the prevention mechanism - confirms the Trivy severity gate
             and branch protection together stop a vulnerable image from reaching
             main.

## 2026-11-14 New Azure subscription: AKS stuck/failed from unregistered resource providers (x5)

Symptom:     Switched to a brand-new Azure subscription (fresh account, no prior
             resource provider registrations). `terraform apply` for the AKS
             foundation and later for Key Vault, Log Analytics, and the
             `oms_agent`/monitoring addon each failed in turn with
             `MissingSubscriptionRegistration` for a different namespace:
             `Microsoft.Compute`, `Microsoft.ContainerService`, `Microsoft.Network`,
             `Microsoft.ContainerRegistry`, `Microsoft.KeyVault`,
             `Microsoft.OperationalInsights`, `Microsoft.OperationsManagement`,
             and finally `microsoft.insights` (hit again later via
             `az aks enable-addons --addons monitoring`).
Looked at:   The exact error names the missing namespace. `az provider show -n
             <Namespace> --query registrationState` confirmed each one as
             `NotRegistered`.
Root cause:  A fresh subscription has none of Azure's resource provider
             namespaces registered. Each Azure service (Compute, AKS, VNets, ACR,
             Key Vault, Log Analytics, Azure Monitor/Insights) needs its own
             provider registered before the first resource of that type can be
             created, and this only surfaces when you actually try to create one.
Fix:         `az provider register --namespace <Namespace>` for each one, then
             poll `az provider show` until `Registered` (took ~1-2 minutes each),
             then re-plan and re-apply.
Prevent:     On any brand-new subscription, register the full provider list
             (`Microsoft.Compute`, `Microsoft.ContainerService`,
             `Microsoft.Network`, `Microsoft.ContainerRegistry`,
             `Microsoft.KeyVault`, `Microsoft.OperationalInsights`,
             `Microsoft.OperationsManagement`, `Microsoft.ManagedIdentity`,
             `microsoft.insights`) up front, before the first `terraform apply`,
             instead of discovering them one at a time mid-apply.

## 2026-11-14 AKS provisioningState stuck on Failed after a mid-update provider error

Symptom:     After the `Microsoft.OperationsManagement` registration error
             interrupted an `azurerm_kubernetes_cluster` update, Terraform's next
             `apply` reported success (`Apply complete!`, outputs printed), but
             `az aks show --query provisioningState` still returned `Failed`, and
             none of the newly-enabled addons (Key Vault CSI driver, Container
             Insights `ama-logs`, Azure Policy/Gatekeeper) had actually been
             scheduled onto the node (`kubectl get pods -A` showed only the
             baseline system pods).
Looked at:   `terraform plan` after the "successful" apply showed `0 to add, 1 to
             change` (a minor, unrelated drain-timeout drift) - the addon fields
             weren't in the diff, so Terraform considered its job done and never
             re-sent them to Azure. `az aks show --query provisioningState` kept
             returning `Failed` even after that apply.
Root cause:  The failed update left AKS's own `provisioningState` stuck, and a
             later Terraform apply that doesn't touch the same fields (because
             Terraform's state already "knows" they're set) never re-triggers
             Azure to retry provisioning them.
Fix:         `az aks disable-addons -a monitoring` then `az aks enable-addons -a
             monitoring --workspace-resource-id <law-id>` (after the `microsoft.
             insights` registration above) - an explicit disable/re-enable cycle
             forces a fresh reconcile attempt, which this time succeeded end to
             end and flipped `provisioningState` back to `Succeeded`.
Prevent:     After any Terraform apply that errors out partway on an Azure
             resource, don't trust `terraform plan` showing "no changes" as proof
             the cluster is healthy - check `az aks show --query provisioningState`
             directly. If it's stuck on `Failed`, force a retry via the specific
             `az aks disable-addons`/`enable-addons` pair rather than waiting for
             an unrelated Terraform diff to happen to touch it.

## 2026-11-14 Single-node lab cluster couldn't fit the app plus the observability stack

Symptom:     After deploying `ledger-api` (2 replicas) on top of the already-
             running AKS system addons (metrics-server, `ama-logs`, CoreDNS,
             workload-identity webhook, Gatekeeper/Azure Policy), both pods stuck
             in `Pending` with `FailedScheduling: 0/1 nodes are available: 1
             Insufficient cpu`. `kubectl describe node` also showed a live
             `OOMKilling` event against Grafana, which had just been installed.
Looked at:   `kubectl describe node` -> Allocated resources showed CPU requests
             already at 99% *before* `ledger-api` was added. Breaking it down per
             pod (`kubectl get pods -A -o json`, summed `resources.requests.cpu`)
             showed the biggest consumers were AKS's own system pods, not
             anything we chose to install: `metrics-server` (2x156m),
             `ama-logs` (2x150m), `coredns` (2x100m), the workload-identity
             webhook (2x100m), Gatekeeper (3x100m).
Root cause:  A single `Standard_D2as_v7` (2 vCPU) node's allocatable CPU is
             consumed almost entirely by AKS's own mandatory system pods once
             Key Vault CSI, Container Insights, and Azure Policy are all
             enabled - there was never enough headroom left for the app, let
             alone Prometheus/Grafana/Trivy Operator on top.
Fix:         Scaled `default_node_pool.node_count` from 1 to 2 in `infra/aks.tf`
             (an in-place Terraform update, no node pool replacement) - cheapest
             fix that stayed well within the subscription's 10-vCPU family quota
             and 10-vCPU total regional cap.
Prevent:     Don't assume a 1-node "cheapest possible" AKS lab cluster has spare
             capacity for anything beyond the bare system addons. Budget for AKS's
             own addon overhead (easily 1-1.5 vCPU on a 2-vCPU node) before sizing
             for the actual workload, especially once Container Insights, Key
             Vault CSI, and Azure Policy are all turned on together.

## 2026-11-14 Grafana OOM-killed at startup with a 256Mi memory limit

Symptom:     Even after scaling to 2 nodes, Grafana's pod was stuck in
             `CrashLoopBackOff`. `kubectl describe pod` showed `Last State:
             Terminated, Reason: OOMKilled, Exit Code: 137` during startup,
             before it ever became ready.
Looked at:   Grafana's own startup logs showed it installing a long list of
             bundled datasource plugins (`grafana-postgresql-datasource`,
             `stackdriver`, `loki`, `tempo`, `cloudwatch`, and about a dozen
             more) during boot, before the process had even reached a steady
             state.
Root cause:  The lab's deliberately tight `limits.memory: 256Mi` for Grafana
             (set to keep the footprint small) was below what current Grafana
             actually needs just to finish its plugin-install boot sequence, let
             alone run afterwards.
Fix:         Raised Grafana's `resources.requests.memory` to 256Mi and
             `limits.memory` to 512Mi in
             `deploy/observability/kube-prometheus-stack-values.yaml`, then
             `helm upgrade`.
Prevent:     Don't size a Helm chart's memory limit purely off "what the running
             process should need" - check what it needs to *start*, especially
             for anything that installs plugins or runs migrations on boot.

## 2026-11-14 ServiceMonitor created no scrape target: Service needs its own `app` label, not just `selector.app`

Symptom:     `ledger-api`'s `ServiceMonitor` and its NetworkPolicy allow-rule were
             both correctly applied, Prometheus's generated config clearly
             contained a `serviceMonitor/ledger/ledger-api/0` job, yet
             `/api/v1/targets` never listed it as active - `ledger-api` never
             showed up, not even as "down". It appeared only in
             `/api/v1/targets?state=any` under `droppedTargets`, with a fully
             populated, correct-looking `discoveredLabels` block (right pod IP,
             right port, right metrics path).
Looked at:   Pulled the live scrape config from Prometheus's own
             `/api/v1/status/config` HTTP endpoint (the only way to see it - the
             `prometheus` container is distroless with no `cat`/`ls`/shell, so
             `kubectl exec ... cat ...` fails silently with an "executable not
             found" error that's easy to misread as "empty/no match" when piped
             through `grep`). The generated `relabel_configs` for the job
             included: `source_labels: [__meta_kubernetes_service_label_app,
             __meta_kubernetes_service_labelpresent_app], regex: (ledger-api);
             true, action: keep`.
Root cause:  Prometheus Operator turns a `ServiceMonitor`'s `selector.matchLabels`
             into a relabel-time `keep` rule against the **Service object's own
             metadata labels** - not the Pod's labels, and not the Service's
             `spec.selector`. `deploy/helm/ledger-api/templates/service.yaml` set
             `spec.selector.app: ledger-api` (which correctly wires the Service to
             the right Pods) but had no `metadata.labels` at all, so
             `__meta_kubernetes_service_labelpresent_app` was `false` and the keep
             rule silently dropped every target.
Fix:         Added `metadata.labels: {app: ledger-api}` to the Service, matching
             what the ServiceMonitor's selector expects, then `helm upgrade`.
Prevent:     A ServiceMonitor's `selector` must match the **Service's own
             labels**, not its `spec.selector` (which only controls which Pods
             the Service routes to) and not the Pod's labels directly. When a
             target silently never appears (not even unhealthy), check
             `droppedTargets` via `?state=any` and read the relabel_configs from
             `/api/v1/status/config` rather than assuming it's a NetworkPolicy or
             RBAC problem.

## 2026-11-14 Kyverno v1.20+ removed execution of the legacy `kyverno.io/v1 ClusterPolicy` type

Symptom:     Applying the three Kyverno policies from PROJECT-PLAN.md's Phase 5
             (`kyverno.io/v1 ClusterPolicy` with `validate`/`verifyImages` rules)
             all failed at `kubectl apply` with: "Kyverno v1.20 removed execution
             of legacy kyverno.io policy types and no longer accepts create or
             update requests for them ... migrate to ValidatingPolicy,
             MutatingPolicy, GeneratingPolicy or ImageValidatingPolicy
             (policies.kyverno.io)".
Looked at:   `kubectl api-resources | grep kyverno` showed both the old
             `kyverno.io/v1 ClusterPolicy` (listed as a real, readable resource
             type) and the new CEL-based `policies.kyverno.io/v1` types
             (`ValidatingPolicy`, `ImageValidatingPolicy`,
             `NamespacedValidatingPolicy`, `NamespacedImageValidatingPolicy`,
             etc.) side by side, and `kubectl explain` against the new types to
             get their real field names (CEL `validations[].expression`,
             `matchConstraints.resourceRules`, `attestors[].cosign.keyless`).
Root cause:  The project plan was written assuming the legacy declarative
             `validate`/`verifyImages` rule style; the Helm chart's current
             `kyverno/kyverno` has moved entirely to Kubernetes
             ValidatingAdmissionPolicy-style CEL expressions and no longer runs
             the old type at all (read/list/delete only).
Fix:         Rewrote all three as `NamespacedValidatingPolicy` (the two plain
             validations) and `NamespacedImageValidatingPolicy` (the signature
             check), scoped to the `ledger` namespace, using CEL expressions
             (`object.spec.containers.all(c, ...)`) instead of JSON-pattern
             `validate` blocks.
Prevent:     Don't trust a project plan's exact manifest syntax for a fast-moving
             admission-control project without checking `kubectl api-resources`
             and `kubectl explain` against what's actually installed first -
             Kyverno's own migration guide
             (https://kyverno.io/docs/guides/migration-to-cel/) says this
             removal is permanent going forward, not a deprecation warning.

## 2026-11-14 Kyverno CEL image-signature expression: wrong syntax for referencing an attestor by name

Symptom:     After fixing the ClusterPolicy->CEL migration above,
             `verify-image-signature` (Audit mode) was accepted by the API server
             but every evaluation recorded `result: error` in its
             `EphemeralReport`, not `pass` or `fail`: "unsupported native
             conversion from string to 'v1alpha1.Attestor'".
Looked at:   The CEL expression was
             `verifyImageSignatures(image, ['github-actions-keyless'])` - passing
             the attestor's *name* as a bare string.
Root cause:  `verifyImageSignatures()` expects actual Attestor objects, exposed
             through a CEL variable named `attestors` (one property per declared
             attestor, by name), not a list of name strings.
Fix:         Changed the expression to
             `verifyImageSignatures(image, [attestors.github_actions_keyless])`
             (and renamed the attestor from `github-actions-keyless` to
             `github_actions_keyless`, since CEL property access escapes `-` to
             `__dash__`, which is uglier than just avoiding dashes in the name).
Prevent:     When a Kyverno CEL policy's `EphemeralReport` shows `result: error`
             (not `fail`), that's a CEL evaluation error in the policy itself, not
             a real validation failure - check the report's `message` field for
             a CEL type error before assuming the image/resource is actually
             non-compliant.

## 2026-11-14 Kyverno image-signature check can't authenticate to our own private ACR

Symptom:     After fixing the CEL syntax above, `verify-image-signature` still
             recorded `result: error` on a real pod using our own ACR image -
             this time: "DefaultAzureCredential: failed to acquire a token ...
             WorkloadIdentityCredential: no client ID specified ...
             ManagedIdentityCredential: failed to authenticate a system assigned
             identity. The endpoint responded with ... Multiple user assigned
             identities exist, please specify the clientId ... AzureCLICredential:
             fork/exec /bin/sh: permission denied".
Looked at:   The `credentials.providers: [azure]` option exists on
             `ImageValidatingPolicy`/`NamespacedImageValidatingPolicy`, but
             Kyverno's own admission-controller pod has no workload identity
             wired up (no `azure.workload.identity/client-id` annotation on its
             ServiceAccount, no federated identity credential, no
             `azure.workload.identity/use: "true"` pod label) - the VM's IMDS
             does have multiple user-assigned identities attached (the AKS
             addons' own identities plus the kubelet identity), so
             `ManagedIdentityCredential` can't guess which one to use, and
             there's no shell in Kyverno's container for `AzureCLICredential`'s
             fallback to even attempt.
Root cause:  Reading a private registry's manifest/signature data (as opposed to
             just pulling the image, which the kubelet's own `AcrPull` role
             already covers) needs Kyverno's *own* pod identity to have registry
             read access - that was never wired up.
Fix:         Not fixed. Documented as a known gap (`docs/security.md`) instead of
             spending the time to give Kyverno's admission controller its own
             federated identity credential + AcrPull role: the policy runs in
             Audit mode specifically because Phase 2b (cosign signing) was never
             implemented, so even with working registry credentials the
             signature check would always fail (there is no signature to find).
             Fixing the credential wiring without also doing the signing work
             would not change the practical outcome.
Prevent:     If `verify-image-signature` is ever flipped to `Enforce`, wire a
             dedicated workload identity for Kyverno's admission-controller
             ServiceAccount (its own federated credential + `AcrPull` on the
             target registry) *and* implement Phase 2b cosign signing in CI first
             - doing either alone doesn't get you a working check.

## 2026-11-14 Trivy Operator's own scanJobsConcurrentLimit setting was silently ignored (wrong values.yaml nesting, twice)

Symptom:     Despite setting `operator.scanJobsConcurrentLimit: 1` in
             `deploy/observability/trivy-operator-values.yaml`, multiple scan
             Jobs still ran at once and several failed with "Failed to acquire
             cache or database lock ... cache may be in use by another process:
             timeout" - Trivy's shared vulnerability DB cache being hit by
             concurrent scanners.
Looked at:   `kubectl get configmap trivy-operator-config -n trivy-system -o
             yaml` showed `OPERATOR_CONCURRENT_SCAN_JOBS_LIMIT: "10"` (the
             chart's default) even after the `helm upgrade`. Misread
             `helm show values aqua/trivy-operator`'s indentation once while
             investigating and "fixed" it to a root-level `scanJobsConcurrentLimit:
             1` (removing the `operator:` nesting) - that made it worse, since
             `helm get values` then showed the root-level key as the live
             release's user-supplied values, meaning the apply had "succeeded"
             while doing nothing at all to the ConfigMap either time.
Root cause:  First pass: a values key nested one level too deep gets silently
             dropped by Helm (no error - it just isn't a recognised path in the
             chart's templates, so nothing renders). Second pass: overcorrected
             based on a misread of the default values dump's indentation and
             moved it OUT of the `operator:` block it actually belongs under.
Fix:         Confirmed the real path with `helm show values aqua/trivy-operator`
             cross-checked against where the `operator:` block starts and ends
             (`awk` on non-indented lines to find the next top-level key), put
             `scanJobsConcurrentLimit` back under `operator:`, `helm upgrade`,
             then verified directly against the ConfigMap
             (`OPERATOR_CONCURRENT_SCAN_JOBS_LIMIT: "1"`) rather than trusting the
             `helm upgrade` output. Also had to `kubectl delete jobs -n
             trivy-system --all` and restart the operator pod to clear the
             already-queued jobs from the wrong-concurrency config before the
             fix actually took visible effect.
Prevent:     Never trust that a `helm upgrade` "succeeded" message means a
             specific value actually landed where intended - check the
             generated ConfigMap/Secret/Deployment directly. When second-
             guessing a values path from a `helm show values` dump, verify the
             parent block's start/end lines first, don't eyeball indentation
             under time pressure.

## 2026-11-14 Trivy Operator's scan jobs can't authenticate to our own private ACR either

Symptom:     Even with concurrency fixed, the `ledger-api` image's
             `VulnerabilityReport` never appeared. The operator's logs showed:
             "UNAUTHORIZED: authentication required" against
             `acrledgerlab.azurecr.io`, after Trivy's `remote` backend (and the
             docker/containerd/podman backends, which don't even apply inside
             the scan job's container) all failed to find credentials.
Looked at:   `helm show values aqua/trivy-operator` for a private-registry
             option: `operator.privateRegistryScanSecretsNames` (a
             namespace->secret-name map) plus
             `operator.accessGlobalSecretsAndServiceAccount` exist for exactly
             this. The fix needs a Kubernetes `docker-registry` secret holding
             real ACR pull credentials - least-privilege via `az acr token
             create` with a read-only scope-map, not the registry-wide admin
             account (which is deliberately disabled, `admin_enabled = false` in
             `infra/acr.tf`).
Root cause:  Same root cause as the Kyverno image-signature gap documented
             above: anything that needs to *read* image data from our ACR
             (rather than just have the kubelet pull it, which already works via
             the `AcrPull` role on `kubelet_identity`) needs its own registry
             credentials, and neither Kyverno's nor Trivy Operator's own pod
             identity had any wired up.
Fix:         Not fixed in this session - creating the ACR token
             (`az acr token create`) was blocked by this environment's own
             command-approval guardrails (a credential-creation action), and
             routing that through the user mid-session wasn't worth it for one
             image's scan when the operator's broader value - continuous
             scanning of the whole cluster, catching real CVEs - was already
             proven (13 VulnerabilityReports generated across AKS system images,
             Gatekeeper, Prometheus, with 12 real HIGH and 12 MEDIUM findings).
Prevent:     Before deploying Trivy Operator (or any in-cluster scanner) against
             a private, admin-disabled ACR, create the read-only ACR token and
             wire `privateRegistryScanSecretsNames` up front, in the same change
             that disables ACR admin access - don't discover the gap after the
             fact per-tool (this is the second time this exact pattern showed
             up this session, see the Kyverno entry above).
