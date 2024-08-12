resource "irmc-redfish_storage_volume" "vol" {
  server {
    username     = ""
    password     = ""
    endpoint     = ""
    ssl_insecure = false
  }

  storage_controller_id = "0"
  raid_type             = "RAID0"
  capacity_bytes        = 100000000
  name                  = "abc"
  init_mode             = "Fast"
  physical_drives       = ["[\"0\", \"3\"]"]
  optimum_io_size_bytes = 131072
  read_mode             = "ReadAhead"
  write_mode            = "WriteThrough"
}
