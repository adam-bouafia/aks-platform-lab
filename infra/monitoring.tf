resource "azurerm_log_analytics_workspace" "main" {
  name                = "law-ledger-lab"
  location            = azurerm_resource_group.main.location
  resource_group_name = azurerm_resource_group.main.name
  sku                 = "PerGB2018"
  retention_in_days   = 30
  daily_quota_gb      = 1 # cost cap for a lab cluster
}
