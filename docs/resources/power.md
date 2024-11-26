<!--
Copyright (c) 2024 Fsas Technologies Inc., or its subsidiaries. All Rights Reserved.

Licensed under the Mozilla Public License Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://mozilla.org/MPL/2.0/

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
-->

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
