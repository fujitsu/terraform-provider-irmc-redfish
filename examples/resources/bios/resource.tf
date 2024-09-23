resource "irmc-redfish_bios" "bio" {
  for_each = var.rack1
  server {
    username     = each.value.username
    password     = each.value.password
    endpoint     = each.value.endpoint
    ssl_insecure = each.value.ssl_insecure
  }

  attributes = {
    "AssetTag" : "MyTag"
    "BIOSParameterBackup" : "Enabled"
  }
  system_reset_type = "ForceRestart"
}
