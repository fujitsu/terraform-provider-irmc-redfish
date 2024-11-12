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
	"fmt"
	"terraform-provider-irmc-redfish/internal/models"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/stmcginnis/gofish/redfish"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &PowerResource{}

func NewPowerResource() resource.Resource {
	return &PowerResource{}
}

// PowerResource defines the resource implementation.
type PowerResource struct {
	p *IrmcProvider
}

const HOST_POWER_ACTION_ENDPOINT = "/redfish/v1/Systems/0/Actions/Oem/FTSComputerSystem.Reset"

func (*PowerResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "power"
}

// PowerSchema to design the schema for power resource.
func PowerResourceSchema() map[string]schema.Attribute {
	const waitTime = 120
	return map[string]schema.Attribute{
		"id": schema.StringAttribute{
			MarkdownDescription: "ID of the power resource",
			Description:         "ID of the power resource",
			Computed:            true,
		},
		"host_power_action": schema.StringAttribute{
			MarkdownDescription: "IRMC Power settings - Applicable values are 'On','ForceOn','ForceOff','ForceRestart'," +
				"'GracefulRestart','GracefulShutdown','PowerCycle', 'PushPowerButton', 'Nmi'",
			Description: "IRMC Power settings - Applicable values are 'On','ForceOn','ForceOff','ForceRestart'," +
				"'GracefulRestart','GracefulShutdown','PowerCycle', 'PushPowerButton', 'Nmi'",
			Required: true,
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.RequiresReplaceIfConfigured(),
			},
			Validators: []validator.String{
				stringvalidator.OneOf(
					string(redfish.OnResetType),
					string(redfish.ForceOnResetType),
					string(redfish.ForceOffResetType),
					string(redfish.ForceRestartResetType),
					string(redfish.GracefulRestartResetType),
					string(redfish.GracefulShutdownResetType),
					string(redfish.PushPowerButtonResetType),
					string(redfish.PowerCycleResetType),
					string(redfish.NmiResetType),
				),
			},
		},

		"max_wait_time": schema.Int64Attribute{
			MarkdownDescription: "The maximum duration in seconds to wait for the server to achieve the desired power state before aborting.",
			Description:         "The maximum duration in seconds to wait for the server to achieve the desired power state before aborting.",
			Computed:            true,
			Optional:            true,
			Default:             int64default.StaticInt64(waitTime),
		},

		"power_state": schema.StringAttribute{
			MarkdownDescription: "IRMC Power State -  might take values: 'On', 'Off'.",
			Description:         "IRMC Power State -  might take values: 'On', 'Off'",
			Computed:            true,
		},
	}
}

func (r *PowerResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "IRMC Host Power resource",
		Description:         "IRMC Host Power resource",
		Attributes:          PowerResourceSchema(),
		Blocks:              RedfishServerResourceBlockMap(),
	}
}

func (r *PowerResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// Create creates the resource and sets the initial Terraform state.
func (r *PowerResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	// Decode the plan data
	tflog.Info(ctx, "resource-power: create starts")

	var powerPlan models.PowerResourceModel
	diags := req.Plan.Get(ctx, &powerPlan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Provide synchronization
	var endpoint string = powerPlan.RedfishServer[0].Endpoint.ValueString()
	var resource_name string = "resource-power"
	mutexPool.Lock(ctx, endpoint, resource_name)
	defer mutexPool.Unlock(ctx, endpoint, resource_name)

	// Initialize the Redfish server connection
	config, err := ConnectTargetSystem(r.p, &powerPlan.RedfishServer)
	if err != nil {
		resp.Diagnostics.AddError("Service Connect Target System Error", err.Error())
		return
	}
	system, err := GetSystemResource(config.Service)
	if err != nil {
		resp.Diagnostics.AddError("Service Get System Resource Error", err.Error())
		return
	}
	powerPlan.Id = types.StringValue(system.ID)

	defer config.Logout()
	var powerErr error

	powerAction := powerPlan.HostPowerAction.ValueString()

	switch powerAction {
	case "On", "ForceOn":
		powerErr = changePowerState(config.Service, true, powerPlan.MaxWaitTime.ValueInt64())

	case "ForceOff":
		powerErr = changePowerState(config.Service, false, powerPlan.MaxWaitTime.ValueInt64())

	case "PowerCycle":
		payload := map[string]string{
			"FTSResetType": "PowerCycle",
		}
		respPost, err := config.Post(HOST_POWER_ACTION_ENDPOINT, payload)
		if err != nil {
			resp.Diagnostics.AddError("PowerCycle POST request failed", err.Error())
			return
		}
		defer respPost.Body.Close()

		if respPost.StatusCode != 204 {
			resp.Diagnostics.AddError("PowerCycle POST request failed - ", fmt.Sprintf("Received status code: %d", respPost.StatusCode))
			return
		}

		powerErr = waitUntilHostStateChanged(config.Service, false, powerPlan.MaxWaitTime.ValueInt64())
		if powerErr != nil {
			resp.Diagnostics.AddError("Host state has not been changed within given timeout", powerErr.Error())
			return
		}
		time.Sleep(30 * time.Second)
	default:
		powerErr = resetHost(config.Service, redfish.ResetType(powerAction),
			powerPlan.MaxWaitTime.ValueInt64())
	}

	if powerErr != nil {
		resp.Diagnostics.AddError("Power Operation Error", powerErr.Error())
		return
	}
	time.Sleep(10 * time.Second)
	powerStateStatus, errpowerstate := isPoweredOn(config.Service)
	if errpowerstate != nil {
		resp.Diagnostics.AddError("Service Connect Target System Error", errpowerstate.Error())
		return
	}
	if powerStateStatus {
		powerPlan.PowerState = types.StringValue("On")
	} else {
		powerPlan.PowerState = types.StringValue("Off")
	}

	tflog.Trace(ctx, "resource-power: create - state update finished")
	diags = resp.State.Set(ctx, &powerPlan)
	resp.Diagnostics.Append(diags...)
	tflog.Info(ctx, "resource-power: create ends")

}

// Read refreshes the Terraform state with the latest data.
func (r *PowerResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	tflog.Info(ctx, "resource-power: read starts")
	var state models.PowerResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Initialize the Redfish server connection
	config, err := ConnectTargetSystem(r.p, &state.RedfishServer)
	if err != nil {
		resp.Diagnostics.AddError("Service Connect Target System Error", err.Error())
		return
	}

	system, err := GetSystemResource(config.Service)
	if err != nil {
		resp.Diagnostics.AddError("system error", err.Error())
		return
	}
	if state.PowerState != types.StringValue(string(system.PowerState)) {
		tflog.Info(ctx, "PowerState different than state, resetting state values.")
		// Workaround for PowerState change when user updates the server in Terraform.
		// The first 'terraform apply' updates the server information.
		// On the next 'terraform apply', Terraform checks PowerState to determine
		// if it should trigger a HostPowerAction (e.g., Power On/Power Off the host).
		state.HostPowerAction = types.StringValue("") //Reset HostPowerAction due to ChangedPowerState
	}

	state.PowerState = types.StringValue(string(system.PowerState))
	tflog.Trace(ctx, "resource_power read: finished reading state")
	// Save into State
	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	tflog.Info(ctx, "resource-power: read ends")
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *PowerResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// Get state Data
	tflog.Info(ctx, "resource-power: update starts")
	var state, plan models.PowerResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Get plan Data
	diags = req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	state.MaxWaitTime = plan.MaxWaitTime
	state.RedfishServer = plan.RedfishServer
	tflog.Trace(ctx, "resource-power: update - state update finished")
	// Save into State
	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	tflog.Info(ctx, "resource-power: update ends")
}

// Delete deletes the resource and removes the Terraform state on success.
func (*PowerResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	tflog.Info(ctx, "resource-power: delete starts")
	resp.State.RemoveResource(ctx)
	tflog.Info(ctx, "resource-power: delete ends")
}
