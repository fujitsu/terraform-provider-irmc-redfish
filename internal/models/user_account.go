package models

import (
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type IrmcUserAccountResourceModel struct {
	Id                            types.String    `tfsdk:"id"`
	RedfishServer                 []RedfishServer `tfsdk:"server"`
	UserID                        types.String    `tfsdk:"user_id"`
	UserUsername                  types.String    `tfsdk:"user_username"`
	UserPassword                  types.String    `tfsdk:"user_password"`
	UserRole                      types.String    `tfsdk:"user_role"`
	UserEnabled                   types.Bool      `tfsdk:"user_enabled"`
	UserRedfishEnabled            types.Bool      `tfsdk:"user_redfish_enabled"`
	UserLanChannelRole            types.String    `tfsdk:"user_lanchannel_role"`
	UserSerialChannelRole         types.String    `tfsdk:"user_serialchannel_role"`
	UserEnabledAccountConfig      types.Bool      `tfsdk:"user_account_config_enabled"`
	UserEnabledIRMCSettingsConfig types.Bool      `tfsdk:"user_irmc_settings_config_enabled"`
	UserEnabledVideoRedirection   types.Bool      `tfsdk:"user_video_redirection_enabled"`
	UserEnabledRemoteStorage      types.Bool      `tfsdk:"user_remote_storage_enabled"`
	UserShellAccess               types.String    `tfsdk:"user_shell_access"`
	UserEnabledAlertChassisEvents types.Bool      `tfsdk:"user_alert_chassis_events"`
}
