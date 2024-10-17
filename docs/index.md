---
page_title: "irmc-redfish Provider"
subcategory: ""
description: |-
  
---

# irmc-redfish Provider

This manual provides overview about iRMC Redfish Terraform usage in context of single or many systems
under Terraform management.

## Example Usage

The provider can be used with systems where credentials are system-specific as well as in configurations
when many systems share same credentials. The idea has been taken from other provider implementations 
where there is no single endpoint, but every managed system will be represented by specific IP address
and possible unique credentials. At the moment there is no possibility to mix these two approaches.

### Configuration with system-specific credentials

For configurations with many systems and necessity to use credentials specific for them, credentials
have to be configured inside of terraform.tfvars.

terraform.tfvars
```terraform
rack = {
    "system-1" = {
        username = "admin"
        password = "aJ$kL0123Bjf!"
        endpoint = "https://10.172.201.205"
        ssl_insecure = true
    },
    "system-2" = {
        endpoint = "https://10.172.201.136"
        username = "admin"
        password = "adminADMIN123"
        ssl_insecure = true
    },
}
```

variables.tf
```terraform
variable "rack1" {
  type = map(object({
    username     = string
    password     = string
    endpoint     = string
    ssl_insecure = bool
  }))
}
```

resource.tf
```terraform
resource "irmc-redfish_power" "pwr" {
  for_each = var.rack1
  server {
    username = each.value.username
    password = each.value.password
    endpoint     = each.value.endpoint
    ssl_insecure = each.value.ssl_insecure
  }

  host_power_action = "ForceOff"
  max_wait_time = 400
}
```

provider.tf
```terraform
terraform {
  required_providers {
    irmc-redfish = {
      version = "1.0.0"
      source  = "hashicorp/fujitsu/irmc-redfish"
    }
  }
}

provider "irmc-redfish" {}
```

### Configuration with same credentials

For configurations with many systems which share same credentials, they can be defined
inside of provider block in provider.tf, while terraform.tfvars contains only endpoint definitions.

terraform.tfvars
```terraform
rack = {
    "system-1" = {
        endpoint = "https://10.172.201.205"
        ssl_insecure = true
    },
    "system-2" = {
        endpoint = "https://10.172.201.136"
        ssl_insecure = true
    },
}
```

variables.tf
```terraform
variable "rack1" {
  type = map(object({
    endpoint     = string
    ssl_insecure = bool
  }))
}
```

resource.tf
```terraform
resource "irmc-redfish_power" "pwr" {
  for_each = var.rack1
  server {
    endpoint     = each.value.endpoint
    ssl_insecure = each.value.ssl_insecure
  }

  host_power_action = "ForceOff"
  max_wait_time = 400
}
```

provider.tf
```terraform
terraform {
  required_providers {
    irmc-redfish = {
      version = "1.0.0"
      source  = "hashicorp/fujitsu/irmc-redfish"
    }
  }
}

provider "irmc-redfish" {
    username = "admin"
    password = "admin"
}
```

## Schema

### Optional

- `password` (String, Sensitive) Password related to given user name accessing Redfish API
- `username` (String) Username accessing Redfish API
