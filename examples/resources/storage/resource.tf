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

resource "irmc-redfish_storage" "storage" {
  for_each = var.rack1
  server {
    username     = each.value.username
    password     = each.value.password
    endpoint     = each.value.endpoint
    ssl_insecure = each.value.ssl_insecure
  }

  job_timeout = 180
  storage_controller_serial_number = "SKC4910421"
  bios_continue_on_error = "PauseOnErrors"
  bios_status = true
  patrol_read = "Manual"
  patrol_read_rate = 32
  bgi_rate = 31
  mdc_rate = 32
  rebuild_rate = 32
  migration_rate = 35
}
