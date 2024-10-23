---
page_title: "irmc-redfish_power Resource - irmc-redfish"
subcategory: ""
description: |-
  IRMC Host Power resource
---

# irmc-redfish_power (Resource)

IRMC Host Power resource.


## Schema

### Required

- `host_power_action` (String) IRMC Power settings - Applicable values are 'On', 'ForceOn', 'ForceOff', 'ForceRestart', 'GracefulRestart', 'GracefulShutdown', 'PowerCycle', 'PushPowerButton', 'Nmi'.

### Optional

- `max_wait_time` (Number) The maximum duration in seconds to wait for the server to achieve the desired power state before aborting (in case of powering on understood as exit of BIOS POST phase).
- `server` (Block List) List of server BMCs and their respective user credentials (see [below for nested schema](#nestedblock--server))

### Read-Only

- `id` (String) ID of the power resource
- `power_state` (String) IRMC Power State -  might take values: 'On', 'Off'.

<a id="nestedblock--server"></a>
### Nested Schema for `server`

Required:

- `endpoint` (String) Server BMC IP address or hostname

Optional:

- `password` (String, Sensitive) User password for login
- `ssl_insecure` (Boolean) This field indicates whether the SSL/TLS certificate must be verified or not
- `username` (String) User name for login
