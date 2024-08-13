// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"strings"

	"terraform-provider-irmc-redfish/internal/models"

	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/stmcginnis/gofish"
	"github.com/stmcginnis/gofish/redfish"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &BootOrderResource{}
var _ resource.ResourceWithImportState = &BootOrderResource{}

func NewBootOrderResource() resource.Resource {
    return &BootOrderResource{}
}

// BootOrderResource defines the resource implementation.
type BootOrderResource struct {
    p *IrmcProvider
}

func (r *BootOrderResource) updateVirtualMediaState(response *redfish.VirtualMedia, plan models.BootOrderResourceModel) models.BootOrderResourceModel {
    var new_id strings.Builder
    new_id.WriteString(VMEDIA_ENDPOINT)
    new_id.WriteString(response.ID)

    return models.BootOrderResourceModel{
        Id: types.StringValue(new_id.String()),
        RedfishServer: plan.RedfishServer,
    }
}

func (r *BootOrderResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + vmediaName
}

func BootOrderSchema() map[string]schema.Attribute {
    return map[string]schema.Attribute{
        "id": schema.StringAttribute{
            Computed:            true,
            MarkdownDescription: "ID of BIOS settings resource on iRMC.",
            Description: "ID of BIOS settings resource on iRMC.",
        },
        "boot_order": schema.ListAttribute{
            Required:            true,
            MarkdownDescription: "Boot devices order in BIOS.",
            Description:         "Boot devices order in BIOS.",
            ElementType:         types.StringType,
            Validators: []validator.List{
                listvalidator.SizeAtLeast(1),
            },
        },
        "system_reset_type": schema.StringAttribute{
            Required: true,
            MarkdownDescription: "Control over if and how system will be reset to finish boot order change.",
            Description: "Control over if and how system will be reset to finish boot order change.",
            Validators: []validator.String{
                stringvalidator.OneOf([]string{
                    "ForceRestart", // TODO: to replace once power state control will be implemented
                    "GracefulRestart",
                    "PowerCycle",
                    "NoRestart",
                }...),
            },
        },
        "system_reset_timeout": schema.Int64Attribute{
            Optional: true,
            Default:  int64default.StaticInt64(600),
            Description: "",
            MarkdownDescription: "",
        },
    }
}

func (r *BootOrderResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
    resp.Schema = schema.Schema{
        MarkdownDescription: "The resource is used to control (read or modify) boot order settings on Fujitsu server equipped with iRMC controller.",
        Description:         "The resource is used to control (read or modify) boot order settings on Fujitsu server equipped with iRMC controller.",
        Attributes:          BootOrderSchema(),
        Blocks:              RedfishServerResourceBlockMap(),
    }
}

func (r *BootOrderResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func applyBootOrderPlan(ctx context.Context, service *gofish.Service, plan models.BootOrderResourceModel) (diags diag.Diagnostics) {
    return diags
}

func validateBootOrderPlan(ctx context.Context, service *gofish.Service, plan models.BootOrderResourceModel) (diags diag.Diagnostics) {
    return diags
}

func (r *BootOrderResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
    tflog.Info(ctx, "boot_order: create starts")

    // Read Terraform plan data into the model
    var plan models.BootOrderResourceModel
    diags := req.Plan.Get(ctx, &plan)
    resp.Diagnostics.Append(diags...)
    if resp.Diagnostics.HasError() {
        return
    }

    // Connect to service
    api, err := ConnectTargetSystem(r.p, &plan.RedfishServer)
    if err != nil {
        resp.Diagnostics.AddError("service error: ", err.Error())
        return
    }

    defer api.Logout()

    // Fetch current boot order and check if planned boot order
    // contains all requested devices
    diags = validateBootOrderPlan(ctx, api.Service, plan)
    resp.Diagnostics.Append(diags...)
    if diags.HasError() {
        return
    }

    // Apply boot order change
    diags = applyBootOrderPlan(ctx, api.Service, plan)
    resp.Diagnostics.Append(diags...)
    if diags.HasError() {
        return
    }

    // TODO: state
    tflog.Info(ctx, "boot_order: create ends")
}

func (r *BootOrderResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
    tflog.Info(ctx, "boot_order: read starts")

    // Read Terraform prior state data into the model
    var state models.BootOrderResourceModel
    resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
    if resp.Diagnostics.HasError() {
        return
    }

    // Connect to service
    api, err := ConnectTargetSystem(r.p, &state.RedfishServer)
    if err != nil {
        resp.Diagnostics.AddError("service error: ", err.Error())
        return
    }

    defer api.Logout()

    // Get information about virtual media slot into which the plan has been applied
    virtualMedia, err := redfish.GetVirtualMedia(api.Service.GetClient(), state.Id.ValueString())
    if err != nil {
        resp.Diagnostics.AddError("Virtual media does not exist: ", err.Error())
        return
    }

    if len(virtualMedia.Image) == 0 {
        return
    }

    // Save updated data into Terraform state
    new_state := r.updateVirtualMediaState(virtualMedia, state)
    resp.Diagnostics.Append(resp.State.Set(ctx, &new_state)...)
    tflog.Info(ctx, "boot_order: read ends")
}

func (r *BootOrderResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
    tflog.Info(ctx, "boot_order: update starts")

    // Read Terraform plan
    var plan models.BootOrderResourceModel
    resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
    if resp.Diagnostics.HasError() {
        return
    }
   
    // Read terraform state
    var state models.BootOrderResourceModel
    diags := req.State.Get(ctx, &state)
    resp.Diagnostics.Append(diags...)
    if resp.Diagnostics.HasError() {
        return
    }

    tflog.Info(ctx, "boot_order: update ends")
}

func (r *BootOrderResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
    tflog.Info(ctx, "boot_order: delete starts")
    resp.State.RemoveResource(ctx)
    tflog.Info(ctx, "boot_order: delete ends")
}

func (r *BootOrderResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
    tflog.Info(ctx, "boot_order: import starts")

    // TODO

    tflog.Info(ctx, "boot_order: import ends")
}
