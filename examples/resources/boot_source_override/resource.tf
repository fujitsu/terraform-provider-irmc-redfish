resource "irmc-redfish_boot_source_override" "bso" {
  for_each = var.rack1
  server {
    username     = each.value.username
    password     = each.value.password
    endpoint     = each.value.endpoint
    ssl_insecure = each.value.ssl_insecure
  }

  boot_source_override_enabled = "Continues"
  boot_source_override_target  = "Hdd"

  system_reset_type = "ForceRestart"
}
