/*
Copyright (c) 2025 Fsas Technologies Inc., or its subsidiaries. All Rights Reserved.

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
	"strconv"
	"strings"

	"terraform-provider-irmc-redfish/internal/models"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/mapvalidator"
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

const IRMC_ATTRIBUTES_SETTINGS_ENDPOINT = "/redfish/v1/Managers/iRMC/Oem/ts_fujitsu/iRMCConfiguration/Attributes"

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &IrmcAttributesResource{}
var _ resource.ResourceWithImportState = &IrmcAttributesResource{}

func NewIrmcAttributesResource() resource.Resource {
	return &IrmcAttributesResource{}
}

// IrmcAttributesResource defines the resource implementation.
type IrmcAttributesResource struct {
	p *IrmcProvider
}

func (r *IrmcAttributesResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + iRMCAttributes
}

func IrmcAttributesSchema() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"id": schema.StringAttribute{
			Computed:            true,
			MarkdownDescription: "ID of iRMC attributes settings resource on iRMC.",
			Description:         "ID of iRMC attributes settings resource on iRMC.",
		},
		"attributes": schema.MapAttribute{
			Required:            true,
			MarkdownDescription: "Map of iRMC attributes.",
			Description:         "Map of iRMC attributes.",
			ElementType:         types.StringType,
			Validators: []validator.Map{
				mapvalidator.SizeAtLeast(1),
			},
		},
		"job_timeout": schema.Int64Attribute{
			Computed:            true,
			Optional:            true,
			Default:             int64default.StaticInt64(600),
			Description:         "Timeout in seconds for iRMC attributes settings change to finish.",
			MarkdownDescription: "Timeout in seconds for iRMC attributes settings change to finish.",
			Validators: []validator.Int64{
				int64validator.AtLeast(240),
			},
		},
	}
}

func (r *IrmcAttributesResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "The resource is used to control (read, modify or import) iRMC attributes settings on Fujitsu server equipped with iRMC controller.",
		Description:         "The resource is used to control (read, modify or import) iRMC attributes settings on Fujitsu server equipped with iRMC controller.",
		Attributes:          IrmcAttributesSchema(),
		Blocks:              RedfishServerResourceBlockMap(),
	}
}

func (r *IrmcAttributesResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *IrmcAttributesResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	tflog.Info(ctx, "resource-irmc-attributes: create starts")

	// Read Terraform plan data into the model
	var plan models.IrmcAttributesResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Provide synchronization
	var endpoint string = plan.RedfishServer[0].Endpoint.ValueString()
	var resource_name string = "resource-irmc-attributes"
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

	adjustedAttributes, diags := validateAndAdjustPlannedIrmcAttributes(ctx, api.Service, plannedAttributes)
	resp.Diagnostics.Append(diags...)
	if diags.HasError() {
		return
	}

	diags, location := applyIrmcAttributes(api.Service, adjustedAttributes)
	resp.Diagnostics.Append(diags...)
	if diags.HasError() {
		return
	}

	diags = waitTillIrmcAttributesSettingsApplied(ctx, api.Service, location, plan.JobTimeout.ValueInt64())
	resp.Diagnostics.Append(diags...)
	if diags.HasError() {
		return
	}

	plan.Id = types.StringValue(IRMC_ATTRIBUTES_SETTINGS_ENDPOINT)

	diags = resp.State.Set(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Info(ctx, "resource-irmc-attributes: create ends")
}

func (r *IrmcAttributesResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	tflog.Info(ctx, "resource-irmc-attributes: read starts")

	// Read Terraform prior state data into the model
	var state models.IrmcAttributesResourceModel
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

	diags := readIrmcAttributesSettingsToModel(ctx, api.Service, &state.Attributes, false)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)

	tflog.Info(ctx, "resource-irmc-attributes: read ends")
}

func (r *IrmcAttributesResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	tflog.Info(ctx, "resource-irmc-attributes: update starts")

	// Read Terraform plan
	var plan models.IrmcAttributesResourceModel
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

	var plannedAttributes map[string]string
	diags = plan.Attributes.ElementsAs(ctx, &plannedAttributes, true)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	adjustedAttributes, diags := validateAndAdjustPlannedIrmcAttributes(ctx, api.Service, plannedAttributes)
	resp.Diagnostics.Append(diags...)
	if diags.HasError() {
		return
	}

	diags, location := applyIrmcAttributes(api.Service, adjustedAttributes)
	resp.Diagnostics.Append(diags...)
	if diags.HasError() {
		return
	}

	diags = waitTillIrmcAttributesSettingsApplied(ctx, api.Service, location, plan.JobTimeout.ValueInt64())
	resp.Diagnostics.Append(diags...)
	if diags.HasError() {
		return
	}

	plan.Id = types.StringValue(IRMC_ATTRIBUTES_SETTINGS_ENDPOINT)

	diags = resp.State.Set(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Info(ctx, "resource-irmc-attributes: update ends")
}

func (r *IrmcAttributesResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	tflog.Info(ctx, "resource-irmc-attributes: delete starts")
	resp.State.RemoveResource(ctx)
	tflog.Info(ctx, "resource-irmc-attributes: delete ends")
}

func (r *IrmcAttributesResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	tflog.Info(ctx, "resource-irmc-attributes: import starts")

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

	tflog.Info(ctx, "resource-irmc-attributes: import ends")
}

type irmcAttributesConfig struct {
	Attributes redfish.SettingsAttributes `json:"Attributes"`
}

func getIrmcAttributesResource(service *gofish.Service) (irmcAttributesConfig, error) {
	res, err := service.GetClient().Get(IRMC_ATTRIBUTES_SETTINGS_ENDPOINT)
	var resource irmcAttributesConfig
	if err != nil {
		return resource, fmt.Errorf("Could not access iRMC attributes resource due to %s", err.Error())
	}

	defer res.Body.Close()

	bodyBytes, err := io.ReadAll(res.Body)
	if err != nil {
		return resource, fmt.Errorf("Error while reading iRMC attributes response body: %s", err.Error())
	}

	err = json.Unmarshal(bodyBytes, &resource)
	if err != nil {
		return resource, fmt.Errorf("Error while iRMC attributes body unmarshalling: %s", err.Error())
	}

	return resource, nil
}

// validateAndAdjustPlannedIrmcAttributes compares planned attributes values with current attributes from system
// pointed by service. Function returns list of applicable attributes after validation.
func validateAndAdjustPlannedIrmcAttributes(ctx context.Context, service *gofish.Service, plannedAttributes map[string]string) (adjustedAttributes map[string]interface{}, diags diag.Diagnostics) {
	resource, err := getIrmcAttributesResource(service)
	if err != nil {
		diags.AddError("Error while reading /iRMCConfiguration/Attributes", err.Error())
		return adjustedAttributes, diags
	}

	if len(resource.Attributes) == 0 {
		diags.AddError("System does not contain any configurable settings", "")
		return adjustedAttributes, diags
	}

	// Since Attributes in Redfish API have different types than string only, they must be unified to map of strings
	// to be easily handled and compared with planned attributes
	currAttributes := convertRedfishAttributesToUnifiedFormat(resource.Attributes)

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
		diags.AddError("Empty list of valid & different attributes to be applied", "List of attributes is empty")
	}

	adjustedAttributes = newAttributes
	return adjustedAttributes, diags
}

// readIrmcAttributesSettingsToModel reads target bios settings from service into state attributes.
func readIrmcAttributesSettingsToModel(ctx context.Context, service *gofish.Service, attrMap *types.Map, updateAll bool) (diags diag.Diagnostics) {
	resource, err := getIrmcAttributesResource(service)
	if err != nil {
		diags.AddError("Error while reading /iRMCConfiguration/Attributes", err.Error())
		return diags
	}

	if len(resource.Attributes) == 0 {
		diags.AddError("System does not contain any configurable settings", "Verify if used iRMC version supports Attributes")
		return diags
	}

	attributesIntoModel := make(map[string]attr.Value)

	attributes := convertRedfishAttributesToUnifiedFormat(resource.Attributes)
	configuredAttributes := attrMap.Elements()
	for key, val := range attributes {
		if updateAll {
			attributesIntoModel[key] = types.StringValue(val)
		} else {
			if _, ok := configuredAttributes[key]; ok {
				// only these attributes are put into the state, which were previously configured by user
				attributesIntoModel[key] = types.StringValue(val)
			}
		}
	}

	*attrMap, diags = types.MapValueFrom(ctx, types.StringType, attributesIntoModel)
	return diags
}

func applyIrmcAttributes(service *gofish.Service, attributes map[string]interface{}) (diags diag.Diagnostics, location string) {
	client := service.GetClient()
	res, err := client.Get(IRMC_ATTRIBUTES_SETTINGS_ENDPOINT)
	if err != nil {
		diags.AddError("Reading /redfish/v1/Managers/iRMC/Oem/ts_fujitsu/iRMCConfiguration/Attributes failed", err.Error())
		return diags, ""
	}

	res.Body.Close()

	payload := map[string]interface{}{
		"Attributes": attributes,
	}

	res, err = client.PatchWithHeaders(IRMC_ATTRIBUTES_SETTINGS_ENDPOINT, payload,
		map[string]string{HTTP_HEADER_IF_MATCH: res.Header.Get(HTTP_HEADER_ETAG)})

	if err != nil {
		diags.AddError("Changing /redfish/v1/Managers/iRMC/Oem/ts_fujitsu/iRMCConfiguration/Attributes failed", err.Error())
		return diags, ""
	}

	res.Body.Close()

	if res.StatusCode == http.StatusAccepted {
		location = res.Header.Get(HTTP_HEADER_LOCATION)
	}
	return diags, location
}

func waitTillIrmcAttributesSettingsApplied(ctx context.Context, service *gofish.Service, task_location string, timeout int64) (diags diag.Diagnostics) {
	_, err := WaitForRedfishTaskEnd(ctx, service, task_location, timeout)
	if err != nil {
		diags.AddError("Task for patching attributes reported error", err.Error())
		logs, internal_diags := FetchRedfishTaskLog(service, task_location)
		if logs == nil {
			diags = append(diags, internal_diags...)
		} else {
			diags.AddError("Task logs for patching attributes", string(logs))
		}
	} else {
		diags = verifyErrorsInIrmcAttributesTaskLog(service, task_location)
	}

	return diags
}

type taskLog struct {
	Messages []struct {
		Time    string `json:"Time"`
		Message string `json:"Message"`
	} `json:"Messages"`
}

func verifyErrorsInIrmcAttributesTaskLog(service *gofish.Service, task_location string) (diags diag.Diagnostics) {
	logs_bytes, internal_diags := FetchRedfishTaskLog(service, task_location)
	if logs_bytes == nil {
		diags = append(diags, internal_diags...)
	} else {
		var config taskLog
		err := json.Unmarshal(logs_bytes, &config)
		if err != nil {
			diags.AddError("Task logs could not be unmarshalled", err.Error())
			return diags
		}

		for _, v := range config.Messages {
			if strings.Contains(v.Message, "Error") {
				diags.AddError("Task log contains error message(s)", v.Message)
			}
		}

	}

	return diags
}
