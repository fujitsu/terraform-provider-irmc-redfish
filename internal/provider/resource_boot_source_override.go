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
	"io"
	"net/http"

	"terraform-provider-irmc-redfish/internal/models"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/stmcginnis/gofish"
	"github.com/stmcginnis/gofish/redfish"
)

type bootSourceOverrideEndpoints struct {
	bootConfigOemEndpoint string
}

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &BootSourceOverrideResource{}

func NewBootSourceOverrideResource() resource.Resource {
	return &BootSourceOverrideResource{}
}

// BootSourceOverrideResource defines the resource implementation.
type BootSourceOverrideResource struct {
	p *IrmcProvider
}

func (r *BootSourceOverrideResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + bootSourceOverrideName
}

func BootSourceOverrideSchema() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"id": schema.StringAttribute{
			Computed:            true,
			MarkdownDescription: "ID of boot source override resource resource on iRMC.",
			Description:         "ID of boot source override resource resource on iRMC.",
		},
		"boot_source_override_target": schema.StringAttribute{
			Required:            true,
			MarkdownDescription: "Requested boot source override target device instead of normal boot device.",
			Description:         "Requested boot source override target device instead of normal boot device.",
			Validators: []validator.String{
				stringvalidator.OneOf([]string{
					"Pxe",
					"Cd",
					"Hdd",
					"BiosSetup",
				}...),
			},
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.RequiresReplace(),
			},
		},
		"boot_source_override_enabled": schema.StringAttribute{
			Required:            true,
			MarkdownDescription: "Requested boot source override timeline.",
			Description:         "Requested boot source override timeline.",
			Validators: []validator.String{
				stringvalidator.OneOf([]string{
					"Once",
					"Continues",
				}...),
			},
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.RequiresReplace(),
			},
		},
		"system_reset_type": schema.StringAttribute{
			Required:            true,
			MarkdownDescription: "Control how system will be reset to finish boot source override change (if host is powered on).",
			Description:         "Control how system will be reset to finish boot source override change (if host is powered on).",
			Validators: []validator.String{
				stringvalidator.OneOf([]string{
					"ForceRestart",
					"GracefulRestart",
					"PowerCycle",
				}...),
			},
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.RequiresReplace(),
			},
		},
		"job_timeout": schema.Int64Attribute{
			Computed:            true,
			Optional:            true,
			Default:             int64default.StaticInt64(600),
			Description:         "Timeout in seconds for boot source override change to finish.",
			MarkdownDescription: "Timeout in seconds for boot source override change to finish.",
			Validators: []validator.Int64{
				int64validator.AtLeast(240),
			},
		},
	}
}

func (r *BootSourceOverrideResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "The resource is used to control (read or modify) boot source override settings on Fujitsu server equipped with iRMC controller.",
		Description:         "The resource is used to control (read or modify) boot source override settings on Fujitsu server equipped with iRMC controller.",
		Attributes:          BootSourceOverrideSchema(),
		Blocks:              RedfishServerResourceBlockMap(),
	}
}

func (r *BootSourceOverrideResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *BootSourceOverrideResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	tflog.Info(ctx, "resource-boot_source_override: create starts")

	// Read Terraform plan data into the model
	var plan models.BootSourceOverrideResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Provide synchronization
	var endpoint = plan.RedfishServer[0].Endpoint.ValueString()
	var resource_name = "resource-boot_source_override"
	mutexPool.Lock(ctx, endpoint, resource_name)
	defer mutexPool.Unlock(ctx, endpoint, resource_name)

	// Connect to service
	api, err := ConnectTargetSystem(r.p, &plan.RedfishServer)
	if err != nil {
		resp.Diagnostics.AddError("service error: %s", err.Error())
		return
	}

	defer api.Logout()

	isFsas, err := IsFsasCheck(ctx, api)
	if err != nil {
		resp.Diagnostics.AddError("Vendor Detection Failed", err.Error())
		return
	}

	endp := getBootSourceOverrideEndpoints(isFsas)

	err = bootSourceOverrideApply(api, &plan, endp.bootConfigOemEndpoint)
	if err != nil {
		resp.Diagnostics.AddError("Error reported by apply procedure %s", err.Error())
		return
	}

	resetType := (redfish.ResetType)(plan.SystemResetType.ValueString())
	timeout := plan.JobTimeout.ValueInt64()
	err = resetOrPowerOnHostWithPostCheck(api.Service, resetType, timeout)
	if err != nil {
		resp.Diagnostics.AddError("Error reported by reset procedure %s", err.Error())
		return
	}

	plan.Id = types.StringValue(endp.bootConfigOemEndpoint)

	diags = resp.State.Set(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Info(ctx, "resource-boot_source_override: create ends")
}

func (r *BootSourceOverrideResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	tflog.Info(ctx, "resource-boot_source_override: read starts")
	tflog.Info(ctx, "resource-boot_source_override: read ends")
}

func (r *BootSourceOverrideResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	tflog.Info(ctx, "resource-boot_source_override: update starts")
	tflog.Info(ctx, "resource-boot_source_override: update ends")
}

func (r *BootSourceOverrideResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	tflog.Info(ctx, "resource-boot_source_override: delete starts")
	resp.State.RemoveResource(ctx)
	tflog.Info(ctx, "resource-boot_source_override: delete ends")
}

type bootConfig struct {
	BootDevice          string `json:"BootDevice"`
	NextBootOnlyEnabled bool   `json:"NextBootOnlyEnabled"`
	Etag                string `json:"@odata.etag"`
}

func bootSourceOverrideApply(api *gofish.APIClient, plan *models.BootSourceOverrideResourceModel, bootConfigOemEndpoint string) error {
	resp, err := api.Get(bootConfigOemEndpoint)
	if err != nil {
		return fmt.Errorf("GET on /BootConfig finished with error '%w'", err)
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("GET on /BootConfig finished with status code %d", resp.StatusCode)
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("error during read of /BootConfig GET response body '%w'", err)
	}

	resp.Body.Close()
	var config bootConfig
	if err = json.Unmarshal(bodyBytes, &config); err != nil {
		return fmt.Errorf("error during unmarshal of /BootConfig GET response '%w'", err)
	}

	config.BootDevice = plan.BootSourceOverrideTarget.ValueString()
	if plan.BootSourceOverrideEnabled.ValueString() == "Once" {
		config.NextBootOnlyEnabled = true
	} else {
		config.NextBootOnlyEnabled = false
	}

	headers := map[string]string{HTTP_HEADER_IF_MATCH: config.Etag}
	resp, err = api.PatchWithHeaders(bootConfigOemEndpoint, config, headers)
	if err != nil {
		return fmt.Errorf("error during Patch of /BootConfig '%s'", err.Error())
	}

	resp.Body.Close()
	return nil
}

func getBootSourceOverrideEndpoints(isFsas bool) bootSourceOverrideEndpoints {
	if isFsas {
		return bootSourceOverrideEndpoints{
			bootConfigOemEndpoint: fmt.Sprintf("/redfish/v1/Systems/0/Oem/%s/BootConfig", FSAS),
		}
	} else {
		return bootSourceOverrideEndpoints{
			bootConfigOemEndpoint: fmt.Sprintf("/redfish/v1/Systems/0/Oem/%s/BootConfig", TS_FUJITSU),
		}
	}
}
