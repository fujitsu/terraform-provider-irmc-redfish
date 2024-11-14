---
page_title: "irmc-redfish_simple_update Resource - irmc-redfish"
subcategory: ""
description: |-
  IRMC Simple Update resource for software update operations.
---

# irmc-redfish_simple_update (Resource)

IRMC Simple Update resource for software update operations.


## Schema

### Required

- `transfer_protocol` (String) Protocol for the update. Supported values: http, https, ftp.
- `update_image` (String) URI of the firmware image for update. Example: "10.172.200.100/binaries/binary.zip"

### Optional

- `operation_apply_time` (String) Time to apply the update. Supported values: Immediate, OnReset..
- `server` (Block List) List of server BMCs and their respective user credentials (see [below for nested schema](#nestedblock--server))
- `ume_tool_directory_name` (String) Path to the directory containing the UME tool, used when performing a Simple Update in offline mode.
- `update_timeout` (Number) Maximum duration in seconds to wait for the Simple Update operation to finish before aborting.

### Read-Only

- `id` (String) Simple Update resource ID.

<a id="nestedblock--server"></a>
### Nested Schema for `server`

Required:

- `endpoint` (String) Server BMC IP address or hostname

Optional:

- `password` (String, Sensitive) User password for login
- `ssl_insecure` (Boolean) This field indicates whether the SSL/TLS certificate must be verified or not
- `username` (String) User name for login
