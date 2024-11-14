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
	"terraform-provider-irmc-redfish/internal/models"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/stmcginnis/gofish"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &SimpleUpdateResource{}

func NewSimpleUpdateResource() resource.Resource {
	return &SimpleUpdateResource{}
}

// SimpleUpdateResource defines the resource implementation.
type SimpleUpdateResource struct {
	p *IrmcProvider
}

const SIMPLE_UPDATE_ENDPOINT = "/redfish/v1/UpdateService/Actions/UpdateService.SimpleUpdate"
const UPDATE_SERVICE_ENDPOINT = "/redfish/v1/UpdateService"
const SIMPLE_UPDATE_TIMEOUT = 3000

func (r *SimpleUpdateResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + simpleUpdate
}

func (r *SimpleUpdateResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "IRMC Simple Update resource for software update operations.",
		Description:         "This resource allows for performing software updates on IRMC servers using the Redfish Simple Update method.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Simple Update resource ID.",
				Description:         "Simple Update resource ID.",
				Computed:            true,
			},
			"transfer_protocol": schema.StringAttribute{
				MarkdownDescription: "Protocol for the update. Supported values: http, https, ftp.",
				Description:         "Protocol for the update. Supported values: http, https, ftp.",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.OneOf("http", "https", "ftp"),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"update_image": schema.StringAttribute{
				MarkdownDescription: "URI of the firmware image for update. Example: \"10.172.200.100/binaries/binary.zip\"",
				Description:         "URI of the firmware image for update. Example: \"10.172.200.100/binaries/binary.zip\"",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"operation_apply_time": schema.StringAttribute{
				MarkdownDescription: "Time to apply the update. Supported values: Immediate, OnReset..",
				Description:         "Time to apply the update. Supported values: Immediate, OnReset.",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("Immediate"),
				Validators: []validator.String{
					stringvalidator.OneOf([]string{
						"Immediate",
						"OnReset",
					}...),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"update_timeout": schema.Int64Attribute{
				MarkdownDescription: "Maximum duration in seconds to wait for the Simple Update operation to finish before aborting.",
				Description:         "Maximum duration in seconds to wait for the Simple Update operation to finish before aborting.",
				Computed:            true,
				Optional:            true,
				Default:             int64default.StaticInt64(SIMPLE_UPDATE_TIMEOUT),
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.RequiresReplace(),
				},
			},
			"ume_tool_directory_name": schema.StringAttribute{
				MarkdownDescription: "Path to the directory containing the UME tool, used when performing a Simple Update in offline mode.",
				Description:         "Path to the directory containing the UME tool, used when performing a Simple Update in offline mode.",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("Tools"),
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
		},
		Blocks: RedfishServerResourceBlockMap(),
	}
}

func (r *SimpleUpdateResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *SimpleUpdateResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	tflog.Info(ctx, "resource-simple-update: create starts")

	var plan models.SimpleUpdateResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var endpoint = plan.RedfishServer[0].Endpoint.ValueString()
	const resource_name = "resource-simple-update"
	mutexPool.Lock(ctx, endpoint, resource_name)
	defer mutexPool.Unlock(ctx, endpoint, resource_name)

	config, err := ConnectTargetSystem(r.p, &plan.RedfishServer)
	if err != nil {
		resp.Diagnostics.AddError("Service Connect Target System Error", err.Error())
		return
	}
	defer config.Logout()

	plan.Id = types.StringValue(SIMPLE_UPDATE_ENDPOINT)

	poweredOn, err := isPoweredOn(config.Service)
	if err != nil {
		resp.Diagnostics.AddError("Power state check failed", err.Error())
		return
	}
	err = UpdateUmeToolsDirName(config, plan.UmeToolDirName.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to update SimpleUpdateOfflineToolsDirName", err.Error())
		return
	}
	taskLocation, diags := ConfigSimpleUpd(
		ctx,
		config,
		plan.UpdateImage.ValueString(),
		plan.Protocol.ValueString(),
		plan.OperationTime.ValueString(),
	)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if plan.OperationTime.ValueString() == "OnReset" && poweredOn {
		tflog.Info(ctx, "resource-simple-update: update will apply on next reset, ending create without waiting")
		diags = resp.State.Set(ctx, &plan)
		resp.Diagnostics.Append(diags...)
		return
	}

	err = CheckSimpleUpdateStatus(ctx, config.Service, taskLocation, plan.UpdateTimeout.ValueInt64())
	if err != nil {
		resp.Diagnostics.AddError("Simple Update task did not complete successfully", err.Error())
		return
	}

	diags = resp.State.Set(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	tflog.Info(ctx, "resource-simple-update: create ends")
}

func (r *SimpleUpdateResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	tflog.Info(ctx, "resource-simple-update: read starts")

	var state models.SimpleUpdateResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)

	tflog.Info(ctx, "resource-simple-update: read ends")
}

func (r *SimpleUpdateResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	tflog.Info(ctx, "resource-simple-update: update starts")

	// All attributes require the resource to be replaced, the Update operation is not needed.

	tflog.Info(ctx, "resource-simple-update: update ends")
}

func (r *SimpleUpdateResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	tflog.Info(ctx, "resource-simple-update: delete starts")

	resp.State.RemoveResource(ctx)

	tflog.Info(ctx, "resource-simple-update: delete ends")
}

func CheckSimpleUpdateStatus(ctx context.Context, service *gofish.Service, location string, timeout int64) error {
	finishedSuccessfully, err := WaitForRedfishTaskEnd(ctx, service, location, timeout)
	if err != nil || !finishedSuccessfully {
		taskLog, diags := FetchRedfishTaskLog(service, location)
		if diags.HasError() {
			return fmt.Errorf("simple Update task did not complete successfully: %s", err)
		}
		return fmt.Errorf("simple Update task failed. Details: %s. Task log: %s", err, string(taskLog))
	}
	return nil
}

func ConfigSimpleUpd(ctx context.Context, config *gofish.APIClient, updateImage string, protocol string, applyTime string) (string, diag.Diagnostics) {
	var diags diag.Diagnostics
	fullImageURI := fmt.Sprintf("%s://%s", protocol, updateImage)
	payload := map[string]interface{}{
		"ImageURI":                    fullImageURI,
		"@Redfish.OperationApplyTime": applyTime,
	}

	resp, err := config.Post(SIMPLE_UPDATE_ENDPOINT, payload)
	if err != nil {
		diags.AddError("Simple Update POST request failed", err.Error())
		return "", diags
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusAccepted {
		diags.AddError("Simple Update request not accepted", fmt.Sprintf("unexpected status code: %d", resp.StatusCode))
		return "", diags
	}

	taskLocation := resp.Header.Get("Location")
	if taskLocation == "" {
		diags.AddError("Task Location Missing", "Location header not found in response")
		return "", diags
	}

	return taskLocation, diags
}

type UpdateServiceResponse struct {
	Oem struct {
		TsFujitsu struct {
			SimpleUpdateOfflineToolsDirName string `json:"SimpleUpdateOfflineToolsDirName"`
		} `json:"ts_fujitsu"`
	} `json:"Oem"`
}

func UpdateUmeToolsDirName(apiClient *gofish.APIClient, umeFileDirectory string) error {
	res, err := apiClient.Get(UPDATE_SERVICE_ENDPOINT)
	if err != nil {
		return fmt.Errorf("failed to fetch data from Redfish endpoint: %v", err)
	}
	defer res.Body.Close()

	var updateService UpdateServiceResponse
	decoder := json.NewDecoder(res.Body)
	err = decoder.Decode(&updateService)
	if err != nil {
		return fmt.Errorf("failed to parse JSON response: %v", err)
	}

	currentDirName := updateService.Oem.TsFujitsu.SimpleUpdateOfflineToolsDirName
	if currentDirName == umeFileDirectory {
		return nil
	}

	patchData := map[string]interface{}{
		"Oem": map[string]interface{}{
			"ts_fujitsu": map[string]interface{}{
				"SimpleUpdateOfflineToolsDirName": umeFileDirectory,
			},
		},
	}

	res, err = apiClient.PatchWithHeaders(UPDATE_SERVICE_ENDPOINT, patchData,
		map[string]string{HTTP_HEADER_IF_MATCH: res.Header.Get(HTTP_HEADER_ETAG)})
	if err != nil {
		return fmt.Errorf("failed to send PATCH request: %v", err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK && res.StatusCode != http.StatusNoContent {
		return fmt.Errorf("PATCH request failed with status code: %d", res.StatusCode)
	}

	return nil
}
