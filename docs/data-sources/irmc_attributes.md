---
page_title: "irmc-redfish_irmc_attributes Data Source - irmc-redfish"
subcategory: ""
description: |-
  This datasource is used to query iRMC attributes
---

# irmc-redfish_irmc_attributes (Data Source)

This datasource is used to query iRMC attributes.
To get list of all supported attributes with their types and limitations, please access the following Redfish resource:
/redfish/v1/Registries/ManagerAttributeRegistry/ManagerAttributeRegistry.v1_0_0.json


## Schema

### Optional

- `server` (Block List) List of server BMCs and their respective user credentials (see [below for nested schema](#nestedblock--server))

### Read-Only

- `attributes` (Map of String) Map of iRMC settings attributes.

<a id="nestedblock--server"></a>
### Nested Schema for `server`

Required:

- `endpoint` (String) Server BMC IP address or hostname

Optional:

- `password` (String, Sensitive) User password for login
- `ssl_insecure` (Boolean) This field indicates whether the SSL/TLS certificate must be verified or not
- `username` (String) User name for login
