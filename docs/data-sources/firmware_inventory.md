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

# irmc-redfish_firmware_inventory (Data Source)

Firmware inventory data source

## Schema

### Optional

- `server` (Block List) List of server BMCs and their respective user credentials (see [below for nested schema](#nestedblock--server))

### Read-Only

- `id` (String) ID of the firmware inventory.
- `inventory` (Attributes List) (see [below for nested schema](#nestedatt--inventory))

<a id="nestedblock--server"></a>
### Nested Schema for `server`

Required:

- `endpoint` (String) Server BMC IP address or hostname

Optional:

- `password` (String, Sensitive) User password for login
- `ssl_insecure` (Boolean) This field indicates whether the SSL/TLS certificate must be verified or not
- `username` (String) User name for login


<a id="nestedatt--inventory"></a>
### Nested Schema for `inventory`

Read-Only:

- `health` (String) Health status of the firmware.
- `id` (String) ID of the firmware member.
- `name` (String) Name of the firmware.
- `odata_id` (String) OData ID of the firmware member.
- `software_id` (String) Software ID of the firmware.
- `state` (String) State of the firmware.
- `updateable` (Boolean) Indicates if the firmware is updateable.
- `version` (String) Version of the firmware.
