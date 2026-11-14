
resource "azurerm_kubernetes_cluster" "main" {
  name                = "aks-ledger-lab"
  location            = azurerm_resource_group.main.location
  resource_group_name = azurerm_resource_group.main.name
  dns_prefix          = "aksledgerlab" # must be globally-ish unique, becomes part of the cluster's API server FQDN

  role_based_access_control_enabled = true

  api_server_access_profile {
    authorized_ip_ranges = ["109.94.154.182/32"] # Adam's current public IPv4 - AKS authorized_ip_ranges is IPv4-only (no IPv6 support); update this if the ISP-assigned IP changes
  }

  default_node_pool {
    name           = "system"
    node_count     = 1
    vm_size        = "Standard_D2as_v7"
    vnet_subnet_id = azurerm_subnet.aks.id
  }

  identity {
    type = "SystemAssigned"
  }

  node_provisioning_profile {
    mode = "Manual"
  }

  network_profile {
    network_plugin      = "azure"
    network_plugin_mode = "overlay"        #  Azure CNI Overlay - pods get IPs from pod_cidr below, NOT from the VNet's address space
    network_data_plane  = "cilium"         #    Azure CNI Powered by Cilium
    network_policy      = "cilium"         #    explicit for Trivy/AZU-0043 - network_data_plane alone isn't enough for the static checker
    pod_cidr            = "192.168.0.0/16" #     an overlay range, must NOT overlap the VNet's 10.10.0.0/16 or the service_cidr below  
    service_cidr        = "10.100.0.0/16"  #     cluster-internal Service IPs  
    dns_service_ip      = "10.100.0.10"
  }
}
