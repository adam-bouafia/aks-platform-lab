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
