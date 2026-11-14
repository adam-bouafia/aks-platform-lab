# ADR-003: Azure Policy and Kyverno together

## Status

Accepted.

## Context

The project needs guardrails that reject unsafe workloads at admission time,
and compliance reporting that shows the cluster meets a baseline standard.
AKS offers the Azure Policy add-on (OPA Gatekeeper under the hood) built in.
Kyverno is a separate CNCF project, installed via Helm.

## Decision

Use both, for different jobs:

- **Azure Policy** (already enabled via `azure_policy_enabled = true` in
  `aks.tf`): the built-in initiative *Kubernetes cluster pod security
  restricted standards for Linux-based workloads*, assigned in Audit mode.
  This gives a compliance score in the Azure Portal, which is what a platform
  team reports up to security/compliance stakeholders across many clusters.
  It is slow (evaluation cycles run every ~30 minutes) and its policy
  language (Rego, wrapped in Azure's constraint templates) is not something
  you write day to day.
- **Kyverno** (`deploy/policies/`): fast, synchronous admission control in
  YAML, for things Azure Policy's built-in library does not cover well -
  specifically verifying a cosign keyless signature and blocking `:latest`
  tags. Kyverno gives an immediate `kubectl apply` rejection with a clear
  message, which is what a developer needs at commit time, not a 30-minute
  delayed compliance report.

## Consequences

- Two systems to maintain instead of one, with some overlapping intent
  (both can, in principle, check for `:latest` or resource limits).
- In production, I would push harder to pick one if the team only needs one
  job done. Here the jobs are genuinely different (compliance reporting at
  scale vs. fast developer feedback and signature verification), so running
  both is deliberate, not accidental sprawl.
- `verify-image-signature` in this repo runs in `Audit` mode, not `Enforce`,
  because Phase 2b (cosign signing in CI) was not done in this session - see
  `notes/failures.md`. In production this would be a hard blocker on
  `Enforce` before go-live.
