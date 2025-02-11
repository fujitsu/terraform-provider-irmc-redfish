terraform {
  required_providers {
    irmc-redfish = {
      version = "0.0.1"
      source  = "registry.terraform.io/fujitsu/irmc-redfish"
    }
  }
}

provider "irmc-redfish" {}
