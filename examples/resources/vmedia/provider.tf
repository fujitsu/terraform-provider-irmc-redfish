terraform {
    required_providers {
        irmc-redfish = {
            version = "1.0.0"
            source = "hashicorp/fujitsu/irmc-redfish"
        }
    }
}

provider "irmc-redfish" {}
