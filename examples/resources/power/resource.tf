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

resource "irmc-redfish_power" "pwr" {
  for_each = var.rack1
  server {
    username     = each.value.username
    password     = each.value.password
    endpoint     = each.value.endpoint
    ssl_insecure = each.value.ssl_insecure
  }


  //   |********************|*******************************************************************************************************|
  //   | IRMC Power options |                                           Description                                                 |
  //   |      (string)      |                                                                                                       |
  //   |--------------------|-------------------------------------------------------------------------------------------------------|
  //   | PushPowerButton    | Simulates a short power button press. The action depends on the power button configuration of the OS" |
  //   | On                 | Power on the platform.                                                                                |
  //   | ForceOn            | Immediate system power on.                                                                              |
  //   | GracefulRestart    | Performs a system shutdown and reboots the system.                                                    |
  //   | GracefulShutdown   | Performs a system shutdown and powers off the system.                                                 |
  //   | ForceRestart       | Immediate system reset without system shutdown.                                                       |
  //   | ForceOff           | Immediate system power off without system shutdown.                                                   |
  //   | PowerCycle         | Switches the system off and on again.                                                                 |
  //   | Nmi                | Triggers a (N)on-(M)askable (I)nterrupt (NMI) and halts the system.                                   |
  //   |********************|*******************************************************************************************************|


  host_power_action = "ForceRestart"

  max_wait_time = 150


}
