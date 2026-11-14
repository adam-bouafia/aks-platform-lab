output "cluster_name" {
  value = azurerm_kubernetes_cluster.main.name
}

output "acr_login_server" {
  value = azurerm_container_registry.main.login_server
}

output "key_vault_name" {
  value = azurerm_key_vault.main.name
}

output "ledger_api_identity_client_id" {
  value = azurerm_user_assigned_identity.ledger_api.client_id
}

output "tenant_id" {
  value = data.azurerm_client_config.current.tenant_id
}

output "oidc_issuer_url" {
  value = azurerm_kubernetes_cluster.main.oidc_issuer_url
}
