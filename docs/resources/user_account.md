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

# irmc-redfish_user_account (Resource)

This resource is used to manage user accounts.


## Schema

### Required

- `user_username` (String) The name of the user.

### Optional

- `server` (Block List) List of server BMCs and their respective user credentials (see [below for nested schema](#nestedblock--server))
- `user_account_config_enabled` (Boolean) Specifies if User Account Configuration is enabled for the user. **Note:** This attribute is related to IPMI, and disabling it may restrict some IPMI privileges.
- `user_alert_chassis_events` (Boolean) Specifies if chassis event alerts are enabled for the user.
- `user_enabled` (Boolean) Specifies if user is enabled.
- `user_id` (String) The ID of the user.
- `user_irmc_settings_config_enabled` (Boolean) Specifies if iRMC Settings Configuration is enabled for the user. **Note:** This attribute is related to IPMI, and disabling it may restrict some IPMI privileges.
- `user_lanchannel_role` (String) LAN Channel Privilege of the user. Available values are 'Administrator', 'Operator', 'User', and 'OEM'.
- `user_password` (String, Sensitive) Password of the user.
- `user_redfish_enabled` (Boolean) Specifies if Redfish is enabled for the user.
- `user_remote_storage_enabled` (Boolean) Specifies if Remote Storage permission is enabled for the user. **Note:** This attribute is related to IPMI, and disabling it may restrict some IPMI privileges.
- `user_role` (String) Role of the user. Available values are 'Administrator', 'Operator', and 'ReadOnly'.
- `user_serialchannel_role` (String) Serial Channel Privilege of the user. Available values are 'Administrator', 'Operator', 'User', and 'OEM'.
- `user_shell_access` (String) Specifies the shell access level for the user. Available values are 'RemoteManager' and 'None'.
- `user_video_redirection_enabled` (Boolean) Specifies if Video Redirection permission is enabled for the user. **Note:** This attribute is related to IPMI, and disabling it may restrict some IPMI privileges.

### Read-Only

- `id` (String) The ID of the IRMC resource.

<a id="nestedblock--server"></a>
### Nested Schema for `server`

Required:

- `endpoint` (String) Server BMC IP address or hostname

Optional:

- `password` (String, Sensitive) User password for login
- `ssl_insecure` (Boolean) This field indicates whether the SSL/TLS certificate must be verified or not
- `username` (String) User name for login
