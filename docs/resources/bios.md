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

# irmc-redfish_bios (Resource)

The resource is used to control (read, modify or import) BIOS settings on Fsas server equipped with iRMC controller.


## Schema

### Required

- `attributes` (Map of String) Map of BIOS attributes.
- `system_reset_type` (String) Control how system will be reset to finish BIOS settings change (if host is powered on). Applicable values are: 'ForceRestart', 'GracefulRestart', 'PowerCycle'.

### Optional

- `job_timeout` (Number) Timeout in seconds for BIOS settings change to finish (default 600s).
- `server` (Block List) List of server BMCs and their respective user credentials (see [below for nested schema](#nestedblock--server))

### Read-Only

- `id` (String) ID of BIOS settings resource on iRMC.

<a id="nestedblock--server"></a>
### Nested Schema for `server`

Required:

- `endpoint` (String) Server BMC IP address or hostname

Optional:

- `password` (String, Sensitive) User password for login
- `ssl_insecure` (Boolean) This field indicates whether the SSL/TLS certificate must be verified or not
- `username` (String) User name for login

## Import

The resource supports importing BIOS settings configuration from a server.
Current BIOS settings of a specific server can be obtained using the following endpoint:
- /redfish/v1/Systems/0/Bios

To import BIOS resource, the following syntax is expected to be used:
```shell
terraform import irmc-redfish_bios.bios "{\"id\":\"<odata id of the volume>\",\"username\":\"<username>\",\"password\":\"<password>\",\"endpoint\":\"<endpoint>\",\"ssl_insecure\":<true/false>}"
```

If import will be executed successfully, you should be able to list state of the imported resource.
The following state allowes you to have control over the resource using Terraform.
To modify resource e.g.: change an attribute property volume name, you should fill in resource terraform file and check with terraform apply if any differences
between state and plan are visible beside these ones which are requested.
