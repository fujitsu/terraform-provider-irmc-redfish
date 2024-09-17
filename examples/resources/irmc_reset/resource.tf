resource "irmc-redfish_irmc_reset" "irmc_rst" {
  for_each = var.rack1
  server {
    username     = each.value.username
    password     = each.value.password
    endpoint     = each.value.endpoint
    ssl_insecure = each.value.ssl_insecure
  }

  id = "iRMC"
}
