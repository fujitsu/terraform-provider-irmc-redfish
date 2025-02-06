/*
Copyright (c) 2025 Fsas Technologies Inc., or its subsidiaries. All Rights Reserved.

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

resource "irmc-redfish_irmc_firmware_update" "irmcfu" {
  for_each = var.rack1
  server {
    username     = each.value.username
    password     = each.value.password
    endpoint     = each.value.endpoint
    ssl_insecure = each.value.ssl_insecure
  }
  update_type         = "File"
  tftp_server_addr    = "10.172.181.125"
  tftp_update_file    = "irmc/RX2530M7/RX2530M7_02.58f_sdr03.83.bin"
  irmc_path_to_binary = "/home/polecp/terraform/terraform-irmc-provider/examples/resources/irmc_firmware_update/firmware_upd_file/RX2530M7_02.58c_sdr03.83.bin"

}
