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
	"strconv"

	"terraform-provider-irmc-redfish/internal/models"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/mapvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	tkpath "github.com/hashicorp/terraform-plugin-framework/path"
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
var _ resource.Resource = &BiosResource{}
var _ resource.ResourceWithImportState = &BiosResource{}

func NewBiosResource() resource.Resource {
	return &BiosResource{}
}

// BiosResource defines the resource implementation.
type BiosResource struct {
	p *IrmcProvider
}

func (r *BiosResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + biosName
}

func BiosSchema() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"id": schema.StringAttribute{
			Computed:            true,
			MarkdownDescription: "ID of BIOS settings resource on iRMC.",
			Description:         "ID of BIOS settings resource on iRMC.",
		},
		"attributes": schema.MapAttribute{
			Required:            true,
			MarkdownDescription: "Map of BIOS attributes.",
			Description:         "Map of BIOS attributes.",
			ElementType:         types.StringType,
			Validators: []validator.Map{
				mapvalidator.SizeAtLeast(1),
			},
		},
		"system_reset_type": schema.StringAttribute{
			Required:            true,
			MarkdownDescription: "Control how system will be reset to finish BIOS settings change (if host is powered on).",
			Description:         "Control how system will be reset to finish BIOS settings change (if host is powered on).",
			Validators: []validator.String{
				stringvalidator.OneOf([]string{
					"ForceRestart",
					"GracefulRestart",
					"PowerCycle",
				}...),
			},
		},
		"job_timeout": schema.Int64Attribute{
			Computed:            true,
			Optional:            true,
			Default:             int64default.StaticInt64(600),
			Description:         "Timeout in seconds for BIOS settings change to finish.",
			MarkdownDescription: "Timeout in seconds for BIOS settings change to finish.",
			Validators: []validator.Int64{
				int64validator.AtLeast(240),
			},
		},
	}
}

func (r *BiosResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "The resource is used to control (read, modify or import) BIOS settings on Fujitsu server equipped with iRMC controller.",
		Description:         "The resource is used to control (read, modify or import) BIOS settings on Fujitsu server equipped with iRMC controller.",
		Attributes:          BiosSchema(),
		Blocks:              RedfishServerResourceBlockMap(),
	}
}

func (r *BiosResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *BiosResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	tflog.Info(ctx, "resource-bios: create starts")

	// Read Terraform plan data into the model
	var plan models.BiosResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Provide synchronization
	var endpoint string = plan.RedfishServer[0].Endpoint.ValueString()
	var resource_name string = "resource-bios"
	mutexPool.Lock(ctx, endpoint, resource_name)
	defer mutexPool.Unlock(ctx, endpoint, resource_name)

	// Connect to service
	api, err := ConnectTargetSystem(r.p, &plan.RedfishServer)
	if err != nil {
		resp.Diagnostics.AddError("service error: ", err.Error())
		return
	}

	defer api.Logout()

	var plannedAttributes map[string]string
	diags = plan.Attributes.ElementsAs(ctx, &plannedAttributes, true)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	adjustedAttributes, diags := validateAndAdjustPlannedAttributes(ctx, api.Service, plannedAttributes)
	resp.Diagnostics.Append(diags...)
	if diags.HasError() {
		return
	}

	diags = applyBiosAttributes(api.Service, adjustedAttributes)
	resp.Diagnostics.Append(diags...)
	if diags.HasError() {
		return
	}

	diags = waitTillBiosSettingsApplied(ctx, api.Service, plan.JobTimeout.ValueInt64(),
		redfish.ResetType(plan.SystemResetType.ValueString()))

	resp.Diagnostics.Append(diags...)
	if diags.HasError() {
		return
	}

	plan.Id = types.StringValue(BIOS_SETTINGS_ENDPOINT)

	diags = resp.State.Set(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Info(ctx, "resource-bios: create ends")
}

func (r *BiosResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	tflog.Info(ctx, "resource-bios: read starts")

	// Read Terraform prior state data into the model
	var state models.BiosResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	api, err := ConnectTargetSystem(r.p, &state.RedfishServer)
	if err != nil {
		resp.Diagnostics.AddError("service error: ", err.Error())
		return
	}

	defer api.Logout()

	diags := readBiosAttributesSettingsToModel(ctx, api.Service, &state.Attributes, false)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)

	tflog.Info(ctx, "resource-bios: read ends")
}

func (r *BiosResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	tflog.Info(ctx, "resource-bios: update starts")

	// Read Terraform plan
	var plan models.BiosResourceModel
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

	isFsas, err := IsFsasCheck(ctx, api)

	if err != nil {
		resp.Diagnostics.AddError("Vendor Detection Failed", err.Error())
		return
	}
	endp := getIrmcAttributesEndpoints(isFsas)
	var plannedAttributes map[string]string
	diags = plan.Attributes.ElementsAs(ctx, &plannedAttributes, true)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	adjustedAttributes, diags := validateAndAdjustPlannedIrmcAttributes(ctx, api.Service, plannedAttributes, endp.irmcAttributesSettingsEndpoint)
	resp.Diagnostics.Append(diags...)
	if diags.HasError() {
		return
	}

	diags = applyBiosAttributes(api.Service, adjustedAttributes)
	resp.Diagnostics.Append(diags...)
	if diags.HasError() {
		return
	}

	diags = waitTillBiosSettingsApplied(ctx, api.Service, plan.JobTimeout.ValueInt64(),
		redfish.ResetType(plan.SystemResetType.ValueString()))

	resp.Diagnostics.Append(diags...)
	if diags.HasError() {
		return
	}

	plan.Id = types.StringValue(BIOS_SETTINGS_ENDPOINT)

	diags = resp.State.Set(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Info(ctx, "resource-bios: update ends")
}

func (r *BiosResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	tflog.Info(ctx, "resource-bios: delete starts")
	resp.State.RemoveResource(ctx)
	tflog.Info(ctx, "resource-bios: delete ends")
}

func (r *BiosResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	tflog.Info(ctx, "resource-bios: import starts")

	var config CommonImportConfig
	err := json.Unmarshal([]byte(req.ID), &config)
	if err != nil {
		resp.Diagnostics.AddError("Error while unmarshalling import config", err.Error())
		return
	}

	server := models.RedfishServer{
		User:        types.StringValue(config.Username),
		Password:    types.StringValue(config.Password),
		Endpoint:    types.StringValue(config.Endpoint),
		SslInsecure: types.BoolValue(config.SslInsecure),
	}

	creds := []models.RedfishServer{server}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, tkpath.Root("server"), creds)...)

	tflog.Info(ctx, "resource-bios: import ends")
}

func applyBiosAttributes(service *gofish.Service, adjustedAttributes map[string]interface{}) (diags diag.Diagnostics) {
	client := service.GetClient()
	res, err := client.Get(BIOS_SETTINGS_ENDPOINT)
	if err != nil {
		diags.AddError("Reading /redfish/v1/Systems/0/Bios/Settings failed", err.Error())
		return diags
	}

	res.Body.Close()

	payload := map[string]interface{}{
		"Attributes": adjustedAttributes,
	}

	res, err = client.PatchWithHeaders(BIOS_SETTINGS_ENDPOINT, payload,
		map[string]string{HTTP_HEADER_IF_MATCH: res.Header.Get(HTTP_HEADER_ETAG)})

	if err != nil {
		diags.AddError("Changing /redfish/v1/Systems/0/Bios/Settings failed", err.Error())
		return diags
	}

	res.Body.Close()
	return diags
}

// validateAndAdjustPlannedAttributes compares planned attributes values with current attributes from system
// pointed by service. Function returns list of applicable attributes after validation.
func validateAndAdjustPlannedAttributes(ctx context.Context, service *gofish.Service, plannedAttributes map[string]string) (adjustedAttributes map[string]interface{}, diags diag.Diagnostics) {
	system, err := GetSystemResource(service)
	if err != nil {
		diags.AddError("Error while reading /Systems/0", err.Error())
		return adjustedAttributes, diags
	}

	rBios, err := system.Bios()
	if err != nil {
		diags.AddError("Error while reading /Systems/0/Bios", err.Error())
		return adjustedAttributes, diags
	}

	if len(rBios.Attributes) == 0 {
		diags.AddError("No BIOS data for BIOS attributes yet", rBios.ODataID)
		return adjustedAttributes, diags
	}

	// Since Attributes in Redfish API have different types than string only, they must be unified to map of strings
	// to be easily handled and compared with planned attributes
	currAttributes := convertRedfishAttributesToUnifiedFormat(rBios.Attributes)

	newAttributes := make(map[string]interface{})

	// Loop over map of plannedAttributes, check if they are supported by the system
	// Check if parameter will not require conversion to another type (like integer)
	for key, newVal := range plannedAttributes {
		currVal, ok := currAttributes[key]
		if !ok {
			var msg string = fmt.Sprintf("Attribute '%s' is not supported by the system", key)
			diags.AddError("Not supported attribute", msg)
			return adjustedAttributes, diags
		}

		if !isAttributeSupported(key) {
			var msg string = fmt.Sprintf("Attribute '%s' is not supported by the resource", key)
			diags.AddError("Not supported attribute by the resource", msg)
			return adjustedAttributes, diags
		}

		if currValInt, err := strconv.Atoi(currVal); err == nil {
			// Current attribute value is integer, so new value must be converted to integer as well
			// to be accepted by Redfish API and BIOS
			newValInt, err := strconv.Atoi(newVal)
			if err != nil {
				var msg string = fmt.Sprintf("Attribute '%s' has type int in current Attributes, but new value conversion failed '%s'", key, err.Error())
				diags.AddError("Attribute type conversion error", msg)
				return adjustedAttributes, diags
			}

			if currValInt-newValInt != 0 {
				newAttributes[key] = newValInt
			} else {
				var log string = fmt.Sprintf("Planned attribute '%s' has same value as current one, so omit", key)
				tflog.Info(ctx, log)
			}
		} else {
			if currVal != newVal {
				newAttributes[key] = newVal
			} else {
				var log string = fmt.Sprintf("Planned attribute '%s' has same value as current one, so omit", key)
				tflog.Info(ctx, log)
			}
		}
	}

	if len(newAttributes) == 0 {
		diags.AddError("Empty list of valid attributes to be applied", "List of attributes is empty")
	}

	adjustedAttributes = newAttributes
	return adjustedAttributes, diags
}

// convertRedfishBiosAttributesToUnifiedFormat converts attributes to common map[string]string format.
func convertRedfishAttributesToUnifiedFormat(input redfish.SettingsAttributes) map[string]string {
	attributes := make(map[string]string)
	for key, val := range input {
		if attrValue, ok := val.(string); ok {
			attributes[key] = attrValue
		} else {
			attributes[key] = fmt.Sprintf("%v", val)
		}
	}

	return attributes
}

// isAttributeSupported returns information whether attribute is or is not supported by this endpoint.
func isAttributeSupported(key string) bool {
	// Some parameters due to their complex JSON structure are not supported by this implementation
	if key == "BootSources" || key == PERSISTENT_BOOT_ORDER_KEY {
		return false
	}

	return true
}

// readBiosAttributesSettingsToModel reads target bios settings from service into state attributes.
func readBiosAttributesSettingsToModel(ctx context.Context, service *gofish.Service, attrMap *types.Map, updateAll bool) (diags diag.Diagnostics) {
	system, err := GetSystemResource(service)
	if err != nil {
		diags.AddError("Error while reading /Systems/0", err.Error())
		return diags
	}

	rBios, err := system.Bios()
	if err != nil {
		diags.AddError("Error while reading /Systems/0/Bios", err.Error())
		return diags
	}

	size := len(rBios.Attributes)
	if size == 0 {
		diags.AddError("No BIOS data for BIOS attributes yet", rBios.ODataID)
		return diags
	}

	var log string = fmt.Sprintf("System/0/Bios returned Attributes with %d elements", size)
	tflog.Info(ctx, log)

	attributesIntoModel := make(map[string]attr.Value)

	attributes := convertRedfishAttributesToUnifiedFormat(rBios.Attributes)
	configuredAttributes := attrMap.Elements()
	for key, val := range attributes {
		if isAttributeSupported(key) {
			if updateAll {
				attributesIntoModel[key] = types.StringValue(val)
			} else {
				if _, ok := configuredAttributes[key]; ok {
					// only these attributes are put into the state, which were previously configured by user
					attributesIntoModel[key] = types.StringValue(val)
				}
			}
		}
	}

	*attrMap, diags = types.MapValueFrom(ctx, types.StringType, attributesIntoModel)
	return diags
}
