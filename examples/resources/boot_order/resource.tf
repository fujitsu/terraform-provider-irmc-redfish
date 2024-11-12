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

resource "irmc-redfish_boot_order" "bo" {
  for_each = var.rack1
  server {
    username     = each.value.username
    password     = each.value.password
    endpoint     = each.value.endpoint
    ssl_insecure = each.value.ssl_insecure
  }

  // RX1330M6
  boot_order = ["NIC.LOM.2.3.IPv4PXE", "HD.Emb.0.5", "NIC.LOM.1.2.IPv4PXE"]
  # boot_order = [ "HD.Emb.0.5", "NIC.LOM.1.2.IPv4PXE", "NIC.LOM.2.3.IPv4PXE" ]
  #  boot_order = [ "HD.Emb.0.5", "NIC.LOM.2.3.IPv4PXE", "NIC.LOM.1.2.IPv4PXE" ]
  #  boot_order = ["NIC.LOM.2.3.IPv4PXE", "HD.Emb.0.5", "HD.Emb.0.5"]

  // rx4770m7
  #  boot_order = ["RAID.Slot.4.0", "SATA.Emb.1.1", "RAID.Slot.4.0.EFI_BOOT_BOOTX64", "NIC.LOM.0.1.IPv4.PXE", "NIC.LOM.0.1.IPv6.PXE", "HD.Emb.1.1"]
  #  boot_order = ["SATA.Emb.1.1", "RAID.Slot.4.0", "RAID.Slot.4.0.EFI_BOOT_BOOTX64", "NIC.LOM.0.1.IPv4.PXE", "HD.Emb.1.1", "NIC.LOM.0.1.IPv6.PXE" ]
  #  boot_order = ["SATA.Emb.1.1", "RAID.Slot.4.0", "RAID.Slot.4.0.EFI_BOOT_BOOTX64", "NIC.LOM.0.1.IPv4.PXE", "HD.Emb.1.1", "NIC.LOM.0.1.IPv6.PXE" ]
  #  boot_order = ["RAID.Slot.4.0", "RAID.Slot.4.0.EFI_BOOT_BOOTX64", "HD.Emb.1.1", "NIC.LOM.0.1.IPv6.PXE", "SATA.Emb.1.1", "NIC.LOM.0.1.IPv4.PXE" ]

  // RX2540M7
  #  boot_order = [
  #    "NIC.Slot.1.1.IPv4.PXE",
  #    "NIC.Slot.1.3.IPv4.PXE",
  #    "NIC.Slot.1.2.IPv6.PXE",
  #    "NIC.Slot.1.3.IPv6.PXE",
  #    "NIC.Slot.1.0.IPv4.PXE",
  #    "NIC.LOM.0.1.IPv6.PXE",
  #    "NIC.Slot.1.1.IPv6.PXE",
  #    "NIC.LOM.0.1.IPv4.PXE",
  #    "RAID.Slot.11.1",
  #    "NIC.Slot.1.2.IPv4.PXE",
  #    "NIC.FlexLOM.1.2.IPv6.PXE",
  #    "NIC.FlexLOM.1.2.IPv4.PXE",
  #    "HD.Emb.1.1",
  #    "NIC.FlexLOM.1.1.IPv6.PXE",
  #    "NIC.FlexLOM.1.1.IPv4.PXE",
  #    "SATA.Emb.1.1",
  #    "NIC.Slot.1.0.IPv6.PXE",
  #    "SATA.Emb.1.2",
  #  ]

  system_reset_type = "ForceRestart"
}
