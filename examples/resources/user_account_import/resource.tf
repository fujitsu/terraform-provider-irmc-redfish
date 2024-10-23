resource "irmc-redfish_user_account" "ua" {
  for_each = var.users

  # Required attributes to find user
  # username can be modified
  user_id                        = each.value.user_id
  user_username                  = each.value.username

  # optional attributes with defaults or user-specific values
  # if there included more than 1 user all optional attribute will be set to 
  # every user. To set attributes invidually set attributes per user in terraform.tfvars file
  user_role                      = lookup(each.value, "user_role", "Administrator")
  user_enabled                   = lookup(each.value, "user_enabled", true)
  user_redfish_enabled           = lookup(each.value, "user_redfish_enabled", true)
  user_lanchannel_role           = lookup(each.value, "user_lanchannel_role", "Administrator")
  user_serialchannel_role        = lookup(each.value, "user_serialchannel_role", "Administrator")
  user_account_config_enabled    = lookup(each.value, "user_account_config_enabled", true)
  user_irmc_settings_config_enabled = lookup(each.value, "user_irmc_settings_config_enabled", true)
  user_video_redirection_enabled = lookup(each.value, "user_video_redirection_enabled", true)
  user_remote_storage_enabled    = lookup(each.value, "user_remote_storage_enabled", true)
  user_shell_access              = lookup(each.value, "user_shell_access", "RemoteManager")
  user_alert_chassis_events      = lookup(each.value, "user_alert_chassis_events", false)

  server {
    username     = "admin"
    password     = "admin"
    endpoint     = "https://10.172.201.36"
    ssl_insecure = true
  }
}
