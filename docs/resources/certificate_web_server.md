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

# irmc-redfish_certificate_web_server (Resource)

This resource is used to upload a web server certificate in the iRMC. Please note that the iRMC will reboot as part of this operation.



## Schema

### Required

- `cert_private_key` (String) Path to the private key (.pem file).
- `cert_public_key` (String) Path to the public key (.pem file).

### Optional

- `server` (Block List) List of server BMCs and their respective user credentials (see [below for nested schema](#nestedblock--server))

### Read-Only

- `id` (String) ID of irmc web server certificate resource on iRMC.

<a id="nestedblock--server"></a>
### Nested Schema for `server`

Required:

- `endpoint` (String) Server BMC IP address or hostname

Optional:

- `password` (String, Sensitive) User password for login
- `ssl_insecure` (Boolean) This field indicates whether the SSL/TLS certificate must be verified or not
- `username` (String) User name for login
