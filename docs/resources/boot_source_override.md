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

# irmc-redfish_boot_source_override (Resource)

The resource is used to control (read or modify) boot source override settings on Fujitsu server equipped with iRMC controller.

## Schema

### Required

- `boot_source_override_enabled` (String) Requested boot source override timeline. Applicable values are: 'Once', 'Continues'.
- `boot_source_override_target` (String) Requested boot source override target device instead of normal boot device. Applicable values are: 'Pxe', 'Cd', "Hdd', 'BiosSetup'.
- `system_reset_type` (String) Control how system will be reset to finish boot source override change (if host is powered on). Applicable values are: 'ForceRestart', 'GracefulRestart', 'PowerCycle'.

### Optional

- `job_timeout` (Number) Timeout in seconds for boot source override change to finish (default 600s).
- `server` (Block List) List of server BMCs and their respective user credentials (see [below for nested schema](#nestedblock--server))

### Read-Only

- `id` (String) ID of boot source override resource resource on iRMC.

<a id="nestedblock--server"></a>
### Nested Schema for `server`

Required:

- `endpoint` (String) Server BMC IP address or hostname

Optional:

- `password` (String, Sensitive) User password for login
- `ssl_insecure` (Boolean) This field indicates whether the SSL/TLS certificate must be verified or not
- `username` (String) User name for login
