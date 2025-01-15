---
page_title: "irmc-redfish_irmc_attributes Resource - irmc-redfish"
subcategory: ""
description: |-
  The resource is used to control (read, modify or import) iRMC attributes settings on Fujitsu server equipped with iRMC controller.
---

# irmc-redfish_irmc_attributes (Resource)

The resource is used to control (read, modify or import) iRMC attributes settings on Fujitsu server equipped with iRMC controller.


## Schema

### Required

- `attributes` (Map of String) Map of iRMC attributes.

### Optional

- `job_timeout` (Number) Timeout in seconds for iRMC attributes settings change to finish.
- `server` (Block List) List of server BMCs and their respective user credentials (see [below for nested schema](#nestedblock--server))

### Read-Only

- `id` (String) ID of iRMC attributes settings resource on iRMC.

<a id="nestedblock--server"></a>
### Nested Schema for `server`

Required:

- `endpoint` (String) Server BMC IP address or hostname

Optional:

- `password` (String, Sensitive) User password for login
- `ssl_insecure` (Boolean) This field indicates whether the SSL/TLS certificate must be verified or not
- `username` (String) User name for login
