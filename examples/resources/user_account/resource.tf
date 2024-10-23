resource "irmc-redfish_user_account" "ua" {
  for_each = var.rack1
  server {
    username     = each.value.username
    password     = each.value.password
    endpoint     = each.value.endpoint
    ssl_insecure = each.value.ssl_insecure
  }
  // Required information to create user
  // user_username = "<username>"
  // user_password = "<password>"
  user_username = "Tester_1"
  user_password = "Testtest123!"
  user_role     = "Administrator"
}
