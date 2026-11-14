# Runbook

## Zero to cluster

```bash
cd infra
terraform init
terraform plan -out=tfplan
terraform apply "tfplan"   # you run this - creates billable resources

az aks get-credentials -g rg-ledger-lab -n aks-ledger-lab
kubectl get nodes
```

## Deploy the app

```bash
az acr build --registry acrledgerlab --image ledger-api:v1 -f app/Dockerfile app/

cd deploy
kubectl apply -f namespace.yaml

helm upgrade --install ledger-api ./helm/ledger-api \
  --set workloadIdentityClientId=$(terraform -chdir=../infra output -raw ledger_api_identity_client_id) \
  --set keyVaultName=$(terraform -chdir=../infra output -raw key_vault_name) \
  --set tenantId=$(terraform -chdir=../infra output -raw tenant_id)

kubectl get pods -n ledger -o wide
```

## Deploy policy, observability, vulnerability management

```bash
kubectl apply -f deploy/policies/

helm repo add prometheus-community https://prometheus-community.github.io/helm-charts
helm repo add aqua https://aquasecurity.github.io/helm-charts
helm repo update

helm upgrade --install kube-prometheus-stack prometheus-community/kube-prometheus-stack \
  -n monitoring --create-namespace \
  -f deploy/observability/kube-prometheus-stack-values.yaml

kubectl apply -f deploy/observability/prometheusrule.yaml

helm upgrade --install trivy-operator aqua/trivy-operator \
  -n trivy-system --create-namespace \
  -f deploy/observability/trivy-operator-values.yaml
```

## Troubleshoot

| Symptom | Likely cause | Check |
|---|---|---|
| Pod stuck `ContainerCreating`, mount error on the secrets volume | Role assignment hasn't propagated yet (can take a few minutes), or the federated identity credential subject doesn't match the ServiceAccount | `kubectl describe pod -n ledger <pod>`, check events |
| `ImagePullBackOff` | `AcrPull` role assignment missing or not yet propagated | `kubectl describe pod`, check the `kubelet_identity` has `AcrPull` on the ACR |
| `kubectl` commands hang or ask for a login that never works | `local_account_disabled` was set without `kubelogin` configured (not applicable in this lab - local accounts are intentionally left enabled, see `docs/security.md`) | n/a here, but the fix is `kubelogin convert-kubeconfig -l azurecli` |
| Terraform `MissingSubscriptionRegistration` error | A resource provider namespace isn't registered on this subscription yet (happens a lot on a brand-new subscription) | `az provider register --namespace <Namespace>`, wait, retry `apply` |
| AKS `provisioningState` stuck on `Failed` after an update error | The failed operation leaves the cluster in `Failed` until something re-triggers a successful reconcile | `az aks disable-addons`/`enable-addons` for the specific addon that failed, or re-run `terraform apply` after fixing the root cause |

## Tear down

```bash
cd infra
terraform plan -destroy -out=tfdestroy
terraform apply "tfdestroy"   # you run this - destructive

az resource list -g rg-ledger-lab -o table   # confirm empty
rm -f tfplan tfdestroy
```
