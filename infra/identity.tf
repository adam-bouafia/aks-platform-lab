resource "azurerm_user_assigned_identity" "ledger_api" {
  name                = "id-ledger-api"
  location            = azurerm_resource_group.main.location
  resource_group_name = azurerm_resource_group.main.name
}

resource "azurerm_role_assignment" "ledger_api_kv_reader" {
  scope                = azurerm_key_vault.main.id
  role_definition_name = "Key Vault Secrets User"
  principal_id         = azurerm_user_assigned_identity.ledger_api.principal_id
}

# Lets the AKS kubelet pull images from our own ACR without admin credentials.
resource "azurerm_role_assignment" "kubelet_acr_pull" {
  scope                = azurerm_container_registry.main.id
  role_definition_name = "AcrPull"
  principal_id         = azurerm_kubernetes_cluster.main.kubelet_identity[0].object_id
}

# Workload identity: a pod in namespace "ledger" running as service account "ledger-api"
# can exchange its Kubernetes token for a token as this identity - no secret stored anywhere.
resource "azurerm_federated_identity_credential" "ledger_api" {
  name                      = "fic-ledger-api"
  user_assigned_identity_id = azurerm_user_assigned_identity.ledger_api.id
  audience                  = ["api://AzureADTokenExchange"]
  issuer                    = azurerm_kubernetes_cluster.main.oidc_issuer_url
  subject                   = "system:serviceaccount:ledger:ledger-api"
}
