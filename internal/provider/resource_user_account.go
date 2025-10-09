/*
Copyright (c) 2024 Fsas Technologies Inc., or its subsidiaries. All Rights Reserved.

Licensed under the Mozilla Public License Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://mozilla.org/MPL/2.0/


Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"unicode"

	"terraform-provider-irmc-redfish/internal/models"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/stmcginnis/gofish"
	"github.com/stmcginnis/gofish/redfish"
)

const (
	USER_TYPE_ADMIN          = "Administrator"
	USER_TYPE_OPERATOR       = "Operator"
	USER_TYPE_USER           = "User"
	USER_TYPE_READ_ONLY      = "ReadOnly"
	USER_TYPE_OEM            = "OEM"
	USER_TYPE_REMOTE_MANAGER = "RemoteManager"
	USER_TYPE_NONE           = "None"
)

const (
	minUserNameLength = 1
	maxUserNameLength = 16
	minPasswordLength = 12
	maxPasswordLength = 20
	maxUserID         = 16
	minUserID         = 2
)

type userAccountImportConfig struct {
	Username    string `json:"username"`
	Password    string `json:"password"`
	Endpoint    string `json:"endpoint"`
	SslInsecure bool   `json:"ssl_insecure"`
	UserID      string `json:"user_id"`
}

const USER_ACCOUNT_ENDPOINT = "/redfish/v1/AccountService/Accounts"
const MIN_PASSW_CONDITIONS = 3

type RedfishMethod string

const (
	Create RedfishMethod = "Create"
	Update RedfishMethod = "Update"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &IrmcUserAccountResource{}
var _ resource.ResourceWithImportState = &IrmcUserAccountResource{}

func NewUserAccountResource() resource.Resource {
	return &IrmcUserAccountResource{}
}

// IrmcUserAccountResource defines the resource implementation.
type IrmcUserAccountResource struct {
	p *IrmcProvider
}

func (r *IrmcUserAccountResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + userAccount
}
func (r *IrmcUserAccountResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "This resource is used to manage user accounts.",
		Description:         "This resource is used to manage user accounts.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The ID of the IRMC resource.",
				Description:         "The ID of the IRMC resource.",
				Computed:            true,
			},
			"user_id": schema.StringAttribute{
				MarkdownDescription: "The ID of the user.",
				Description:         "The ID of the user.",
				Optional:            true,
				Computed:            true,
			},
			"user_username": schema.StringAttribute{
				MarkdownDescription: "The name of the user.",
				Description:         "The name of the user.",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.LengthBetween(minUserNameLength, maxUserNameLength),
				},
			},
			"user_password": schema.StringAttribute{
				MarkdownDescription: "Password of the user.",
				Description:         "Password of the user.",
				Optional:            true,
				Computed:            true,
				Sensitive:           true,
			},
			"user_role": schema.StringAttribute{
				MarkdownDescription: "Role of the user. Available values are 'Administrator', 'Operator', and 'ReadOnly'.",
				Description:         "Role of the user. Available values are 'Administrator', 'Operator', and 'ReadOnly'.",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString(USER_TYPE_ADMIN),
				Validators: []validator.String{
					stringvalidator.OneOf([]string{
						USER_TYPE_ADMIN,
						USER_TYPE_OPERATOR,
						USER_TYPE_READ_ONLY,
					}...),
				},
			},
			"user_enabled": schema.BoolAttribute{
				MarkdownDescription: "Specifies if user is enabled.",
				Description:         "Specifies if user is enabled.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(true),
			},
			"user_redfish_enabled": schema.BoolAttribute{
				MarkdownDescription: "Specifies if Redfish is enabled for the user.",
				Description:         "Specifies if Redfish is enabled for the user.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(true),
			},
			"user_lanchannel_role": schema.StringAttribute{
				MarkdownDescription: "LAN Channel Privilege of the user. Available values are 'Administrator', 'Operator', 'User', and 'OEM'.",
				Description:         "LAN Channel Privilege of the user. Available values are 'Administrator', 'Operator', 'User', and 'OEM'.",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString(USER_TYPE_ADMIN),
				Validators: []validator.String{
					stringvalidator.OneOf([]string{
						USER_TYPE_ADMIN,
						USER_TYPE_OPERATOR,
						USER_TYPE_USER,
						USER_TYPE_OEM,
					}...),
				},
			},
			"user_serialchannel_role": schema.StringAttribute{
				MarkdownDescription: "Serial Channel Privilege of the user. Available values are 'Administrator', 'Operator', 'User', and 'OEM'.",
				Description:         "Serial Channel Privilege of the user. Available values are 'Administrator', 'Operator', 'User', and 'OEM'.",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString(USER_TYPE_ADMIN),
				Validators: []validator.String{
					stringvalidator.OneOf([]string{
						USER_TYPE_ADMIN,
						USER_TYPE_OPERATOR,
						USER_TYPE_USER,
						USER_TYPE_OEM,
					}...),
				},
			},
			"user_account_config_enabled": schema.BoolAttribute{
				MarkdownDescription: "Specifies if User Account Configuration is enabled for the user. **Note:** This attribute is related to IPMI, and disabling it may restrict some IPMI privileges.",
				Description:         "Specifies if User Account Configuration is enabled for the user. **Note:** This attribute is related to IPMI, and disabling it may restrict some IPMI privileges.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(true),
			},
			"user_irmc_settings_config_enabled": schema.BoolAttribute{
				MarkdownDescription: "Specifies if iRMC Settings Configuration is enabled for the user. **Note:** This attribute is related to IPMI, and disabling it may restrict some IPMI privileges.",
				Description:         "Specifies if iRMC Settings Configuration is enabled for the user. **Note:** This attribute is related to IPMI, and disabling it may restrict some IPMI privileges.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(true),
			},
			"user_video_redirection_enabled": schema.BoolAttribute{
				MarkdownDescription: "Specifies if Video Redirection permission is enabled for the user. **Note:** This attribute is related to IPMI, and disabling it may restrict some IPMI privileges.",
				Description:         "Specifies if Video Redirection permission is enabled for the user. **Note:** This attribute is related to IPMI, and disabling it may restrict some IPMI privileges.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(true),
			},
			"user_remote_storage_enabled": schema.BoolAttribute{
				MarkdownDescription: "Specifies if Remote Storage permission is enabled for the user. **Note:** This attribute is related to IPMI, and disabling it may restrict some IPMI privileges.",
				Description:         "Specifies if Remote Storage permission is enabled for the user. **Note:** This attribute is related to IPMI, and disabling it may restrict some IPMI privileges.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(true),
			},
			"user_shell_access": schema.StringAttribute{
				MarkdownDescription: "Specifies the shell access level for the user. Available values are 'RemoteManager' and 'None'.",
				Description:         "Specifies the shell access level for the user. Available values are 'RemoteManager' and 'None'.",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString(USER_TYPE_REMOTE_MANAGER),
				Validators: []validator.String{
					stringvalidator.OneOf(
						USER_TYPE_REMOTE_MANAGER,
						USER_TYPE_NONE),
				},
			},
			"user_alert_chassis_events": schema.BoolAttribute{
				MarkdownDescription: "Specifies if chassis event alerts are enabled for the user.",
				Description:         "Specifies if chassis event alerts are enabled for the user.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
			},
		},
		Blocks: RedfishServerResourceBlockMap(),
	}
}

func (r *IrmcUserAccountResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	p, ok := req.ProviderData.(*IrmcProvider)

	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *IrmcProvider, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)

		return
	}

	r.p = p
}

func (r *IrmcUserAccountResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	tflog.Info(ctx, "resource-user-account: create starts")
	// Get Plan Data
	var plan models.IrmcUserAccountResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Provide synchronization
	var endpoint string = plan.RedfishServer[0].Endpoint.ValueString()
	var resource_name string = "resource-user-account"
	mutexPool.Lock(ctx, endpoint, resource_name)
	defer mutexPool.Unlock(ctx, endpoint, resource_name)

	userPassword := plan.UserPassword.ValueString()
	userName := plan.UserUsername.ValueString()
	userId := plan.UserID.ValueString()

	config, err := ConnectTargetSystem(r.p, &plan.RedfishServer)
	if err != nil {
		resp.Diagnostics.AddError("error. Service Connect Target System Error", err.Error())
		return
	}

	defer config.Logout()

	isFsas, err := IsFsasCheck(ctx, config)
	if err != nil {
		resp.Diagnostics.AddError("Vendor Detection Failed", err.Error())
		return
	}
	plan.Id = types.StringValue(USER_ACCOUNT_ENDPOINT)

	// Chec Password validation
	err = CheckPasswordValidation(userPassword)
	if err != nil {
		resp.Diagnostics.AddError("error.", err.Error())
		return
	}

	accountList, err := GetListOfUserAccounts(config.Service)
	if err != nil {
		resp.Diagnostics.AddError("error.", err.Error())
		return
	}

	// check if username is free to use
	err = CheckIsUsernameTaken(accountList, userName)
	if err != nil {
		resp.Diagnostics.AddError("error.", err.Error())
		return
	}

	// check if user id already exists
	err = CheckUserIDExistence(accountList, userId)
	if err != nil {
		resp.Diagnostics.AddError("error.", err.Error())
		return
	}
	createPayload, err := InitializeUserAccountRedfishRequest(plan, Create, isFsas)
	if err != nil {
		resp.Diagnostics.AddError("error.", err.Error())
		return
	}

	url := USER_ACCOUNT_ENDPOINT
	respPost, err := config.Post(url, createPayload)
	if err != nil {
		resp.Diagnostics.AddError("error. creating HTTP request: %v", err.Error())
		return
	}

	defer respPost.Body.Close()

	if respPost.StatusCode != http.StatusCreated {
		resp.Diagnostics.AddError("error. User Account Creation POST request failed - ", fmt.Sprintf("Received status code: %d", respPost.StatusCode))
		return
	}

	accountList, err = GetListOfUserAccounts(config.Service)
	if err != nil {
		resp.Diagnostics.AddError("error.", err.Error())
		return
	}

	userId, err = FindUserIDByName(accountList, userName)
	if err != nil {
		resp.Diagnostics.AddError("error.", err.Error())
		return
	}
	plan.UserID = types.StringValue(userId)
	plan.Id = types.StringValue(fmt.Sprintf("%s/%s", USER_ACCOUNT_ENDPOINT, userId))
	// Save into State
	diags = resp.State.Set(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	tflog.Info(ctx, "resource-user-account: create ends")

}

func (r *IrmcUserAccountResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	tflog.Info(ctx, "resource-user-account: read starts")
	var state models.IrmcUserAccountResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	config, err := ConnectTargetSystem(r.p, &state.RedfishServer)
	if err != nil {
		resp.Diagnostics.AddError("Service Connect Target System Error", err.Error())
		return
	}
	defer config.Logout()

	isFsas, err := IsFsasCheck(ctx, config)
	if err != nil {
		resp.Diagnostics.AddError("Vendor Detection Failed", err.Error())
		return
	}

	userID := state.UserID.ValueString()
	url := fmt.Sprintf("%s/%s", USER_ACCOUNT_ENDPOINT, userID)

	respGet, err := config.Get(url)
	if err != nil {
		if respGet != nil && respGet.StatusCode == http.StatusNotFound {
			tflog.Warn(ctx, fmt.Sprintf("User account with ID %s not found, removing from state.", userID))
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading Redfish user account", err.Error())
		return
	}
	defer respGet.Body.Close()

	var data map[string]interface{}
	err = json.NewDecoder(respGet.Body).Decode(&data)
	if err != nil {
		resp.Diagnostics.AddError("Error decoding JSON from Redfish user account response", err.Error())
		return
	}

	if val, ok := data["Enabled"].(bool); ok {
		state.UserEnabled = types.BoolValue(val)
	}
	if val, ok := data["UserName"].(string); ok {
		state.UserUsername = types.StringValue(val)
	}
	if val, ok := data["RoleId"].(string); ok {
		state.UserRole = types.StringValue(val)
	}
	if state.UserPassword.IsNull() || state.UserPassword.IsUnknown() || state.UserPassword.ValueString() == "" {
		state.UserPassword = types.StringNull()
	}

	var oemKey string
	if isFsas {
		oemKey = FSAS
	} else {
		oemKey = TS_FUJITSU
	}

	if oem, oemOK := data["Oem"].(map[string]interface{}); oemOK {
		if oemData, oemDataOK := oem[oemKey].(map[string]interface{}); oemDataOK {
			if baseValues, ok := oemData["BaseValues"].(map[string]interface{}); ok {
				if val, ok := baseValues["Shell"].(string); ok {
					state.UserShellAccess = types.StringValue(val)
				}
				if val, ok := baseValues["Enabled"].(bool); ok {
					state.UserRedfishEnabled = types.BoolValue(val)
				}
			}
			if permissions, ok := oemData["Permissions"].(map[string]interface{}); ok {
				if standard, ok := permissions["Standard"].(map[string]interface{}); ok {
					if val, ok := standard["Lan"].(string); ok {
						state.UserLanChannelRole = types.StringValue(val)
					}
					if val, ok := standard["Serial"].(string); ok {
						state.UserSerialChannelRole = types.StringValue(val)
					}
				}
				if extended, ok := permissions["Extended"].(map[string]interface{}); ok {
					if val, ok := extended["ConfigureUsers"].(bool); ok {
						state.UserEnabledAccountConfig = types.BoolValue(val)
					}
					if val, ok := extended["ConfigureIrmc"].(bool); ok {
						state.UserEnabledIRMCSettingsConfig = types.BoolValue(val)
					}
					if val, ok := extended["UseVideoRedirection"].(bool); ok {
						state.UserEnabledVideoRedirection = types.BoolValue(val)
					}
					if val, ok := extended["UseRemoteStorage"].(bool); ok {
						state.UserEnabledRemoteStorage = types.BoolValue(val)
					}
				}
			}
			if email, ok := oemData["Email"].(map[string]interface{}); ok {
				if val, ok := email["AlertChassisEventsUser"].(bool); ok {
					state.UserEnabledAlertChassisEvents = types.BoolValue(val)
				}
			}
		}
	}

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)

	tflog.Info(ctx, "resource-user-account: read ends")

}

func (r *IrmcUserAccountResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	tflog.Info(ctx, "resource-user-account: update starts")

	var state models.IrmcUserAccountResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var plan models.IrmcUserAccountResourceModel
	diags = req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	config, err := ConnectTargetSystem(r.p, &state.RedfishServer)
	if err != nil {
		resp.Diagnostics.AddError("Service Connect Target System Error", err.Error())
		return
	}
	defer config.Logout()

	isFsas, err := IsFsasCheck(ctx, config)
	if err != nil {
		resp.Diagnostics.AddError("Vendor Detection Failed", err.Error())
		return
	}

	userID := state.UserID.ValueString()
	if userID == "" {
		resp.Diagnostics.AddError("Missing User ID", "User ID is missing in the current state")
		return
	}

	userPassword := plan.UserPassword.ValueString()
	if userPassword != "" {
		err = CheckPasswordValidation(userPassword)
		if err != nil {
			resp.Diagnostics.AddError("Password validation failed", err.Error())
			return
		}
	}

	updatePayload, err := InitializeUserAccountRedfishRequest(plan, Update, isFsas)
	if err != nil {
		resp.Diagnostics.AddError("Failed to initialize update payload", err.Error())
		return
	}

	url := fmt.Sprintf("%s/%s", USER_ACCOUNT_ENDPOINT, userID)
	tflog.Debug(ctx, fmt.Sprintf("Update URL: %s", url))

	respGet, err := config.Get(url)
	if err != nil {
		resp.Diagnostics.AddError("Error reading Redfish user account", err.Error())
		return
	}
	defer respGet.Body.Close()

	etag := respGet.Header.Get(HTTP_HEADER_ETAG)
	if etag == "" {
		resp.Diagnostics.AddError("Missing ETag", "ETag header is missing in the GET response")
		return
	}

	respPatch, err := config.PatchWithHeaders(url, updatePayload, map[string]string{
		HTTP_HEADER_IF_MATCH: etag,
	})
	if err != nil {
		resp.Diagnostics.AddError("Error sending PATCH request", err.Error())
		return
	}
	defer respPatch.Body.Close()

	if respPatch.StatusCode != http.StatusOK && respPatch.StatusCode != http.StatusNoContent {
		resp.Diagnostics.AddError("User Account Update PATCH request failed", fmt.Sprintf("Received status code: %d", respPatch.StatusCode))
		return
	}
	respGet, err = config.Get(url)
	if err != nil {
		resp.Diagnostics.AddError("error. Not able to read updated Redfish user account", err.Error())
		return
	}
	defer respGet.Body.Close()

	if respGet.StatusCode == http.StatusNotFound {
		resp.State.RemoveResource(ctx)
		return
	} else if respGet.StatusCode != http.StatusOK {
		resp.Diagnostics.AddError("error. Not able to read updated Redfish user account", fmt.Sprintf("Received status code: %d", respGet.StatusCode))
		return
	}

	var data map[string]interface{}
	err = json.NewDecoder(respGet.Body).Decode(&data)
	if err != nil {
		resp.Diagnostics.AddError("error. Decoding JSON from Redfish user account response failed", err.Error())
		return
	}

	if val, ok := data["Enabled"].(bool); ok {
		plan.UserEnabled = types.BoolValue(val)
	}
	if val, ok := data["UserName"].(string); ok {
		plan.UserUsername = types.StringValue(val)
	}
	if val, ok := data["RoleId"].(string); ok {
		plan.UserRole = types.StringValue(val)
	}
	if plan.UserPassword.IsNull() || plan.UserPassword.IsUnknown() || plan.UserPassword.ValueString() == "" {
		plan.UserPassword = types.StringNull()
	}

	var oemKey string
	if isFsas {
		oemKey = FSAS
	} else {
		oemKey = TS_FUJITSU
	}

	if oem, oemOK := data["Oem"].(map[string]interface{}); oemOK {
		if oemData, oemDataOK := oem[oemKey].(map[string]interface{}); oemDataOK {
			if baseValues, ok := oemData["BaseValues"].(map[string]interface{}); ok {
				if val, ok := baseValues["Shell"].(string); ok {
					plan.UserShellAccess = types.StringValue(val)
				}
				if val, ok := baseValues["Enabled"].(bool); ok {
					plan.UserRedfishEnabled = types.BoolValue(val)
				}
			}
			if permissions, ok := oemData["Permissions"].(map[string]interface{}); ok {
				if standard, ok := permissions["Standard"].(map[string]interface{}); ok {
					if val, ok := standard["Lan"].(string); ok {
						plan.UserLanChannelRole = types.StringValue(val)
					}
					if val, ok := standard["Serial"].(string); ok {
						plan.UserSerialChannelRole = types.StringValue(val)
					}
				}
				if extended, ok := permissions["Extended"].(map[string]interface{}); ok {
					if val, ok := extended["ConfigureUsers"].(bool); ok {
						plan.UserEnabledAccountConfig = types.BoolValue(val)
					}
					if val, ok := extended["ConfigureIrmc"].(bool); ok {
						plan.UserEnabledIRMCSettingsConfig = types.BoolValue(val)
					}
					if val, ok := extended["UseVideoRedirection"].(bool); ok {
						plan.UserEnabledVideoRedirection = types.BoolValue(val)
					}
					if val, ok := extended["UseRemoteStorage"].(bool); ok {
						plan.UserEnabledRemoteStorage = types.BoolValue(val)
					}
				}
			}
			if email, ok := oemData["Email"].(map[string]interface{}); ok {
				if val, ok := email["AlertChassisEventsUser"].(bool); ok {
					plan.UserEnabledAlertChassisEvents = types.BoolValue(val)
				}
			}
		}
	}
	plan.UserID = state.UserID
	plan.Id = types.StringValue(fmt.Sprintf("%s/%s", USER_ACCOUNT_ENDPOINT, userID))

	diags = resp.State.Set(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Info(ctx, "resource-user-account: update ends")
}

func (r *IrmcUserAccountResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	tflog.Info(ctx, "resource-user-account: delete starts")

	var state models.IrmcUserAccountResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	config, err := ConnectTargetSystem(r.p, &state.RedfishServer)
	if err != nil {
		resp.Diagnostics.AddError("Service Connect Target System Error", err.Error())
		return
	}
	defer config.Logout()

	userID := state.UserID.ValueString()
	if userID == "" {
		resp.Diagnostics.AddError("Missing User ID", "User ID is missing in the current state")
		return
	}

	url := fmt.Sprintf("%s/%s", USER_ACCOUNT_ENDPOINT, userID)

	respDelete, err := config.Delete(url)
	if err != nil {
		resp.Diagnostics.AddError("Error sending DELETE request", err.Error())
		return
	}
	defer respDelete.Body.Close()

	if respDelete.StatusCode != http.StatusOK && respDelete.StatusCode != http.StatusNoContent {
		resp.Diagnostics.AddError("User Account Delete DELETE request failed", fmt.Sprintf("Received status code: %d", respDelete.StatusCode))
		return
	}

	resp.State.RemoveResource(ctx)

	tflog.Info(ctx, "resource-user-account: delete ends")
}

func (r *IrmcUserAccountResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	tflog.Info(ctx, "resource-user_account: import starts")

	var config userAccountImportConfig

	err := json.Unmarshal([]byte(req.ID), &config)
	if err != nil {
		resp.Diagnostics.AddError("Error while unmarshalling id", err.Error())
	}

	server := models.RedfishServer{
		User:        types.StringValue(config.Username),
		Password:    types.StringValue(config.Password),
		Endpoint:    types.StringValue(config.Endpoint),
		SslInsecure: types.BoolValue(config.SslInsecure),
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("user_id"), config.UserID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("server"), []models.RedfishServer{server})...)

	tflog.Info(ctx, "resource-user_account: import ends")
}

// Function to verify if a username already exists in the account list.
func CheckIsUsernameTaken(accounts []*redfish.ManagerAccount, username string) error {
	for _, account := range accounts {
		if account.UserName == username {
			return fmt.Errorf("the username '%v' is already associated with account ID %v. Please choose a different username", username, account.ID)
		}
	}
	return nil
}

func CheckPasswordValidation(password string) error {
	if len(password) < minPasswordLength || len(password) > maxPasswordLength {
		return fmt.Errorf("password for user must be at least 12 characters long")
	}

	hasLower := false
	hasUpper := false
	hasDigit := false
	hasSpecial := false

	for _, char := range password {
		switch {
		case unicode.IsLower(char):
			hasLower = true
		case unicode.IsUpper(char):
			hasUpper = true
		case unicode.IsDigit(char):
			hasDigit = true
		case strings.ContainsRune("'-!\"#$%&()*,./:;?@[\\]^_`{|}~+<=>", char):
			hasSpecial = true
		}
	}

	passwordConditions := 0
	if hasLower {
		passwordConditions++
	}
	if hasUpper {
		passwordConditions++
	}
	if hasDigit {
		passwordConditions++
	}
	if hasSpecial {
		passwordConditions++
	}

	if passwordConditions < MIN_PASSW_CONDITIONS {
		return fmt.Errorf("fulfill at least 3 conditions for password: at least one lowercase letter, one uppercase letter, one number, and one special character")
	}
	return nil
}

func GetListOfUserAccounts(service *gofish.Service) ([]*redfish.ManagerAccount, error) {
	accountService, err := service.AccountService()
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve account service: %v", err)
	}
	accounts, err := accountService.Accounts()
	if err != nil {
		return nil, fmt.Errorf("error retrieving accounts: %v", err)
	}

	return accounts, nil
}

func CheckUserIDExistence(accounts []*redfish.ManagerAccount, userID string) error {
	// if userID == 0, new account will be created
	if len(userID) == 0 {
		return nil
	}
	for _, acc := range accounts {
		if acc.ID == userID && len(acc.UserName) > 0 {
			return fmt.Errorf("user ID %v is already taken. Please provide a different ID", userID)
		}
	}

	return nil
}

func InitializeUserAccountRedfishRequest(plan models.IrmcUserAccountResourceModel, redfishMethod RedfishMethod, isFsas bool) (map[string]interface{}, error) {
	var oemKey string
	if isFsas {
		oemKey = FSAS
	} else {
		oemKey = TS_FUJITSU
	}

	oemPayload := map[string]interface{}{
		"BaseValues": map[string]interface{}{
			"Enabled": plan.UserRedfishEnabled.ValueBool(),
			"Shell":   plan.UserShellAccess.ValueString(),
		},
		"Permissions": map[string]interface{}{
			"Standard": map[string]interface{}{
				"Lan":    plan.UserLanChannelRole.ValueString(),
				"Serial": plan.UserSerialChannelRole.ValueString(),
			},
			"Extended": map[string]interface{}{
				"ConfigureUsers":      plan.UserEnabledAccountConfig.ValueBool(),
				"ConfigureIrmc":       plan.UserEnabledIRMCSettingsConfig.ValueBool(),
				"UseVideoRedirection": plan.UserEnabledVideoRedirection.ValueBool(),
				"UseRemoteStorage":    plan.UserEnabledRemoteStorage.ValueBool(),
			},
		},
		"Email": map[string]interface{}{
			"AlertChassisEventsUser": plan.UserEnabledAlertChassisEvents.ValueBool(),
		},
	}

	if redfishMethod == Create {
		redfishRequest := map[string]interface{}{
			"UserName": plan.UserUsername.ValueString(),
			"Password": plan.UserPassword.ValueString(),
			"RoleId":   plan.UserRole.ValueString(),
			"Enabled":  plan.UserEnabled.ValueBool(),
			"Oem":      map[string]interface{}{oemKey: oemPayload},
		}
		return redfishRequest, nil

	} else if redfishMethod == Update {
		redfishRequest := map[string]interface{}{
			"UserName": plan.UserUsername.ValueString(),
			"Enabled":  plan.UserEnabled.ValueBool(),
			"RoleId":   plan.UserRole.ValueString(),
			"Oem":      map[string]interface{}{oemKey: oemPayload},
		}
		if !plan.UserPassword.IsNull() && !plan.UserPassword.IsUnknown() && plan.UserPassword.ValueString() != "" {
			redfishRequest["Password"] = plan.UserPassword.ValueString()
		}
		return redfishRequest, nil
	}

	return nil, fmt.Errorf("error. Wrong Redfish method")

}

func FindUserIDByName(accounts []*redfish.ManagerAccount, targetUserName string) (string, error) {
	for _, acc := range accounts {
		if acc.UserName == targetUserName {
			return acc.ID, nil
		}
	}
	return "", fmt.Errorf("user with username '%s' not found", targetUserName)
}
