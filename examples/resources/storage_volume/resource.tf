resource "irmc-redfish_storage_volume" "volume" {
  for_each = var.rack1
  server {
    username     = each.value.username
    password     = each.value.password
    endpoint     = each.value.endpoint
    ssl_insecure = each.value.ssl_insecure
  }

  storage_controller_id = "0" //"PRAID EP640i (0)"
  raid_type             = "RAID0"
  //  capacity_bytes        = 100000000
  name      = "new-volume2"
  init_mode = "Fast"
  #    physical_drives = ["[\"0\", \"3\"]"]
  optimum_io_size_bytes = 131072
  read_mode             = "ReadAhead"
  write_mode            = "WriteThrough"

  #    raid_type = "RAID10"
  physical_drives = ["[\"0\", \"1\"]"]
  #    physical_drives = ["[\"0\", \"1\"]", "[\"3\", \"5\"]"]
  lifecycle {
    ignore_changes = [capacity_bytes, physical_drives]
  }
}
