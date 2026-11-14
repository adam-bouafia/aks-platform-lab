resource "azurerm_resource_group" "main" {
  name     = "rg-ledger-lab"
  location = "germanywestcentral"
}


resource "azurerm_virtual_network" "main" {
  name                = "vnet-ledger-lab"
  address_space       = ["10.10.0.0/16"]
  resource_group_name = azurerm_resource_group.main.name
  location            = azurerm_resource_group.main.location
}



resource "azurerm_subnet" "aks" {
  name                 = "subnet-aks"
  resource_group_name  = azurerm_resource_group.main.name
  virtual_network_name = azurerm_virtual_network.main.name
  address_prefixes     = ["10.10.1.0/24"]
}
