# ADR-002: No ingress controller in this lab

## Status

Accepted.

## Context

community `ingress-nginx` (the most commonly used AKS ingress controller)
was retired in March 2026. AKS's own direction for HTTP routing is the
**Gateway API**, with AKS App Routing moving to an Istio-based managed
Gateway API implementation. For a single internal service accessed only for
demos and testing, standing up a full Gateway API implementation is a
meaningful chunk of extra surface (its own CRDs, its own RBAC, its own
failure modes) for no real benefit.

## Decision

No ingress for `ledger-api`. Access is via `kubectl port-forward` for demos,
which is also how the project's own test traffic and curl checks were run.
If this needed external exposure in production, the AKS-managed
Gateway API add-on (Istio-based) is the current recommended path over
community `ingress-nginx`, which is unmaintained.

## Consequences

- No public endpoint, no TLS termination to configure or demo - acceptable
  for an internal lab service, not acceptable for anything meant to be
  reached from outside the cluster.
- `NetworkPolicy` for the app (`deploy/helm/ledger-api/templates/networkpolicy.yaml`)
  allows ingress only from the `ledger` and `monitoring` namespaces - there is
  no ingress-controller namespace to allow, because there is no ingress
  controller.
- If the Gateway API route is taken later, the NetworkPolicy ingress rule
  needs an additional `namespaceSelector` for whichever namespace the
  Gateway API's data plane runs in.
