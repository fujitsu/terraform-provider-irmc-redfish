resource "irmc-redfish_virtual_media" "vm" {
  for_each = var.rack1
  server {
    username     = each.value.username
    password     = each.value.password
    endpoint     = each.value.endpoint
    ssl_insecure = each.value.ssl_insecure
  }

  image                  = "http://10.172.181.125:8006/gauge/vmedia/Cd!123.iso"
  transfer_protocol_type = "HTTP"
}
