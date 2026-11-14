# ADR-001: Azure CNI Overlay with Cilium

## Status

Accepted.

## Context

AKS offers three main networking choices: kubenet, classic Azure CNI (every
pod gets a real VNet IP), and Azure CNI Overlay. `NetworkPolicy` enforcement
(a hard requirement here - see `deploy/helm/ledger-api/templates/networkpolicy.yaml`)
needs a CNI that implements it; kubenet needs Calico bolted on for that, and
has a documented ~400-node scaling ceiling from its reliance on route tables.
Classic Azure CNI gives every pod a VNet IP, which exhausts a /24-sized
subnet fast and wasn't worth the IP budget for a lab VNet sized at `10.10.0.0/16`
with one `/24` subnet for nodes.

## Decision

Azure CNI Overlay, with Cilium as the dataplane (`network_plugin_mode =
"overlay"`, `network_data_plane = "cilium"`, `network_policy = "cilium"` in
`infra/aks.tf`). Pods get IPs from an overlay range (`pod_cidr =
"192.168.0.0/16"`) that never touches the VNet's address space, and Cilium
enforces `NetworkPolicy` natively, with no extra component to install.

## Consequences

- Pod IPs are not reachable from outside the overlay by other VNet-attached
  resources without extra routing - acceptable here, since nothing outside
  the cluster needs to reach a pod directly (access is via `kubectl
  port-forward` or, in production, a Service/Gateway).
- Cilium gives L3/L4 `NetworkPolicy` enforcement out of the box, which is
  what makes the default-deny network in `deploy/helm/ledger-api/templates/networkpolicy.yaml`
  actually work, not just sit there unenforced.
- No separate CNI add-on to install or upgrade (unlike kubenet+Calico):
  Cilium ships as part of the AKS network profile.
