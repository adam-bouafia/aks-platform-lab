terraform {
  required_version = ">= 1.17.0"
  required_providers {
    azurerm = {
      source  = "hashicorp/azurerm"
      version = "~> 5.0"
    }
  }
}



provider "azurerm" {
  features {}
  subscription_id = var.subscription_id
}
