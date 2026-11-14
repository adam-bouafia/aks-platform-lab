resource "random_string" "kv_suffix" {
  length  = 4
  special = false
  upper   = false
}

resource "azurerm_key_vault" "main" {
  name                = "kv-ledger-${random_string.kv_suffix.result}"
  location            = azurerm_resource_group.main.location
  resource_group_name = azurerm_resource_group.main.name
  tenant_id           = data.azurerm_client_config.current.tenant_id
  sku_name            = "standard"

  rbac_authorization_enabled = true
  purge_protection_enabled   = false # lab only - prod needs this on, see notes/failures.md
}

# Terraform itself needs write access to create the secret below.
resource "azurerm_role_assignment" "kv_terraform_writer" {
  scope                = azurerm_key_vault.main.id
  role_definition_name = "Key Vault Secrets Officer"
  principal_id         = data.azurerm_client_config.current.object_id
}

resource "random_password" "api_key" {
  length  = 32
  special = false
}

resource "azurerm_key_vault_secret" "api_key" {
  name         = "api-key"
  value        = random_password.api_key.result
  key_vault_id = azurerm_key_vault.main.id

  depends_on = [azurerm_role_assignment.kv_terraform_writer]
}
