data "irmc-redfish_virtual_media" "vm" {
  for_each = var.rack1
  server {
    username     = each.value.username
    password     = each.value.password
    endpoint     = each.value.endpoint
    ssl_insecure = each.value.ssl_insecure
  }
}

output "virtual_media" {
  value     = data.irmc-redfish_virtual_media.vm
  sensitive = true
}
