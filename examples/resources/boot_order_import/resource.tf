resource "irmc-redfish_boot_order" "bo" {
  server {
    username     = "admin"
    password     = "adminADMIN123"
    endpoint     = "https://10.172.201.40"
    ssl_insecure = true
  }

  boot_order = []
  system_reset_type = "ForceRestart"
}
