# ADR-004: kube-prometheus-stack plus Azure-native Container Insights

## Status

Accepted.

## Context

The project needs metrics (RED: rate, errors, duration), logs, and an alert
that actually fires. Two paths exist on AKS: the fully Azure-native route
(Azure Monitor managed Prometheus + Azure Managed Grafana + Container
Insights), or the self-managed `kube-prometheus-stack` Helm chart with
Container Insights kept only for logs.

## Decision

Use `kube-prometheus-stack` (Prometheus + Grafana + Alertmanager, Alertmanager
disabled here) for metrics, dashboards and the `PrometheusRule` alert, and
keep the AKS `oms_agent` add-on (Container Insights) for logs and the three
KQL queries in `docs/kql/`. This is the path used in `php-gitops-lab`, so it
reuses what was already learned there, and it is what most teams running
self-managed Kubernetes end up operating day to day.

## Consequences

- Two separate systems for metrics and logs: Prometheus/Grafana for metrics,
  Log Analytics/KQL for logs. A fully Azure-native stack would put both
  behind one pane (Azure Monitor Workbooks), at the cost of being tied to
  Azure's query language and pricing model everywhere.
- Running Prometheus and Grafana as pods costs cluster CPU/memory (kept
  deliberately small in `deploy/observability/kube-prometheus-stack-values.yaml`:
  6h retention, no persistent volumes, light resource requests) and needs
  someone to upgrade the chart.
- **Production trade-off worth naming in the interview:** Azure Monitor
  managed Prometheus + Azure Managed Grafana removes the operational burden
  of running and upgrading Prometheus yourself, and keeps data in Azure's
  region/compliance boundary (data residency), at a direct per-GB-ingested
  cost and less flexibility in scrape config than a self-managed stack. For
  a small platform team without dedicated observability engineers, the
  managed route is usually the better default; for a team that already runs
  Prometheus elsewhere and wants one consistent tool, self-managed wins.
