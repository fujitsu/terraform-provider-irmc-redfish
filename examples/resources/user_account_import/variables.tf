/*
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
*/

variable "servers" {
  description = "Map of Redfish servers to manage"
  type = map(object({
    username     = string
    password     = string
    endpoint     = string
    ssl_insecure = bool
  }))
}

variable "users" {
  type = map(object({
    user_id   = string                            # Required
    username  = string                            # Required
    password  = optional(string)                  # Optional
    user_role = optional(string, "Administrator") # Optional, Default: "Administrator"
    # Available values: "Administrator", "Operator", "ReadOnly"
    user_enabled         = optional(bool, true)              # Optional, Default: true
    user_redfish_enabled = optional(bool, true)              # Optional, Default: true
    user_lanchannel_role = optional(string, "Administrator") # Optional, Default: "Administrator"
    # Available values: "Administrator", "Operator", "User", "OEM"
    user_serialchannel_role = optional(string, "Administrator") # Optional, Default: "Administrator"
    # Available values: "Administrator", "Operator", "User", "OEM"
    user_account_config_enabled       = optional(bool, true)              # Optional, Default: true
    user_irmc_settings_config_enabled = optional(bool, true)              # Optional, Default: true
    user_video_redirection_enabled    = optional(bool, true)              # Optional, Default: true
    user_remote_storage_enabled       = optional(bool, true)              # Optional, Default: true
    user_shell_access                 = optional(string, "RemoteManager") # Optional, Default: "RemoteManager"
    # Available values: "RemoteManager", "None"
    user_alert_chassis_events = optional(bool, false) # Optional, Default: false
  }))
}