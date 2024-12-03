resource "irmc-redfish_storage_volume" "volume" {
  for_each = var.rack1
  server {
    username     = each.value.username
    password     = each.value.password
    endpoint     = each.value.endpoint
    ssl_insecure = each.value.ssl_insecure
  }

  storage_controller_serial_number = "SKC4910421"
  raid_type                        = "RAID0"

  // Optional, if not defined maximum capacity for given RAID raid_type
  // and used physical_drives will be choosen
  capacity_bytes = 100000000000
  name           = "new-volume2"

  init_mode             = "Fast"
  optimum_io_size_bytes = 65536
  read_mode             = "ReadAhead"
  write_mode            = "AlwaysWriteBack"
  drive_cache_mode      = "Enabled"

  physical_drives = ["[\"6\", \"7\"]"]
}
