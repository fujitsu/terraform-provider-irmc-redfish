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
	"io"
	"net/http"
	"terraform-provider-irmc-redfish/internal/models"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/stmcginnis/gofish"
	"github.com/stmcginnis/gofish/redfish"
)

const RESET_TIMEOUT int = 600
const CHECK_INTERVAL int = 10

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &IrmcRestartResource{}

func NewIrmcRestartResource() resource.Resource {
	return &IrmcRestartResource{}
}

// IrmcRestartResource defines the resource implementation.
type IrmcRestartResource struct {
	p *IrmcProvider
}

func (r *IrmcRestartResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + irmcRestart
}

func IrmcRestartSchema() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"id": schema.StringAttribute{
			Optional:            true,
			Computed:            true,
			MarkdownDescription: "ID of irmc reset resource on iRMC.",
			Description:         "ID of irmc reset resource on iRMC.",
		},
	}
}

func (r *IrmcRestartResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "This resource is used to reset the IRMC.",
		Description:         "This resource is used to reset the IRMC.",
		Attributes:          IrmcRestartSchema(),
		Blocks:              RedfishServerResourceBlockMap(),
	}
}

func (r *IrmcRestartResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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
func (r *IrmcRestartResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	tflog.Info(ctx, "resource-irmc-reset: create starts")
	// Get Plan Data
	var plan models.IrmcResetResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var endpoint = plan.RedfishServer[0].Endpoint.ValueString()
	var resource_name = "resource-irmc-reset"
	mutexPool.Lock(ctx, endpoint, resource_name)
	defer mutexPool.Unlock(ctx, endpoint, resource_name)

	config, err := ConnectTargetSystem(r.p, &plan.RedfishServer)
	if err != nil {
		resp.Diagnostics.AddError("Service Connect Target System Error", err.Error())
		return
	}

	defer config.Logout()
	var irmc []*redfish.Manager

	// Get manager
	irmc, err = config.Service.Managers()
	if err != nil {
		resp.Diagnostics.AddError("Error when accessing Managers resource", err.Error())
		return
	}
	plan.Id = types.StringValue(irmc[0].ID)

	// Perform manager reset
	err = irmc[0].Reset(redfish.GracefulRestartResetType)
	if err != nil {
		resp.Diagnostics.AddError("Error resetting manager", err.Error())
		return
	}

	config, err = retryConnectWithTimeout(ctx, r.p, &plan.RedfishServer)
	if err != nil {
		resp.Diagnostics.AddError("Service Connect Target System Error", err.Error())
		return
	}

	err = checkIrmcStatus(ctx, config, CHECK_INTERVAL, RESET_TIMEOUT)
	if err != nil {
		resp.Diagnostics.AddError("Failed to reboot IRMC. The operation may take longer than expected to complete.", err.Error())
		return
	}

	tflog.Info(ctx, "resource-irmc-reset: updating state finished")
	// Save into State
	diags = resp.State.Set(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	tflog.Info(ctx, "resource-irmc-reset: create ends")
}

func (r *IrmcRestartResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	tflog.Info(ctx, "resource-irmc-reset: read starts")
	var state models.IrmcResetResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	tflog.Info(ctx, "resource-irmc-reset: read ends")
}

// Update modifies the resource state but returns an error if triggered, as updates are not supported.
func (*IrmcRestartResource) Update(_ context.Context, _ resource.UpdateRequest, resp *resource.UpdateResponse) {
	// This function should not be called since updates are not supported; the resource should be recreated instead.
	resp.Diagnostics.AddError(
		"Unsupported Update Operation for IRMC Reset",
		"The IRMC Reset resource does not support in-place updates. It is intended to be destroyed and recreated if changes are required.",
	)
}

// Delete deletes the resource and removes the Terraform state on success.
func (*IrmcRestartResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	tflog.Info(ctx, "resource-irmc-reset: delete starts")
	// Delete State Data
	resp.State.RemoveResource(ctx)
	tflog.Info(ctx, "resource-irmc-reset: delete ends")
}

func checkIrmcStatus(ctx context.Context, service *gofish.APIClient, interval int, timeout int) error {
	path := "/redfish/v1/"

	time.Sleep(45 * time.Second)

	for start := time.Now(); time.Since(start) < (time.Duration(timeout) * time.Second); {
		tflog.Info(ctx, "Checking IRMC server status via API GET")

		resp, err := service.Get(path)
		if err != nil {
			tflog.Warn(ctx, fmt.Sprintf("GET on %s reported error: %s", path, err.Error()))
			time.Sleep(time.Duration(interval) * time.Second)
			continue
		}

		if resp.StatusCode == http.StatusOK {
			defer CloseResource(resp.Body)
			_, err := io.ReadAll(resp.Body)
			if err != nil {
				tflog.Error(ctx, fmt.Sprintf("Error reading response body: %s", err.Error()))
				return err
			}
			return nil
		}

		tflog.Warn(ctx, fmt.Sprintf("Received non-200 status code: %d", resp.StatusCode))

		time.Sleep(time.Duration(interval) * time.Second)
	}

	return fmt.Errorf("IRMC server status check timed out after %d seconds", timeout)
}
