resource "irmc-redfish_storage_volume" "volume" {
  for_each = var.rack1
  server {
    username     = each.value.username
    password     = each.value.password
    endpoint     = each.value.endpoint
    ssl_insecure = each.value.ssl_insecure
  }

  storage_controller_id = "1" //"PRAID EP640i (0)"
  raid_type             = "RAID0"
  capacity_bytes        = 2000000000
  name                  = "new-volume"
  init_mode             = "Fast"
  optimum_io_size_bytes = 65536
  read_mode             = "ReadAhead"
  write_mode            = "WriteThrough"
  cache_mode            = "Direct"

  physical_drives = ["[\"0\", \"1\"]"]
  lifecycle {
    ignore_changes = [capacity_bytes, physical_drives]
  }
}
