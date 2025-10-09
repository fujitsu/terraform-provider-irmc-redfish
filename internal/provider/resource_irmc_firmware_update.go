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
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"terraform-provider-irmc-redfish/internal/models"
	"terraform-provider-irmc-redfish/internal/validators"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/stmcginnis/gofish"
	"github.com/stmcginnis/gofish/redfish"
)

const (
	FIRMWARE_UPDATE_TIMEOUT = 3000
	UPDATE_TYPE             = "update_type"
	UPDATE_TYPE_FILE        = "File"
	UPDATE_TYPE_TFTP        = "TFTP"
	UPDATE_TYPE_MEMORY_CARD = "MemoryCard"
)

type firmwareUpdateEndpoints struct {
	FirmwareUpdateEndpoint           string
	FileFirmwareUpdateEndpoint       string
	TftpFirmwareUpdateEndpoint       string
	MemoryCardFirmwareUpdateEndpoint string
}

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &IrmcFirmwareUpdateResource{}

func NewIrmcFirmwareUpdateResource() resource.Resource {
	return &IrmcFirmwareUpdateResource{}
}

// IrmcFirmwareUpdateResource defines the resource implementation.
type IrmcFirmwareUpdateResource struct {
	p *IrmcProvider
}

func (r *IrmcFirmwareUpdateResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + firmwareUpdate
}
func IrmcFirmwareUpdateSchema() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"id": schema.StringAttribute{
			MarkdownDescription: "ID of the IRMC firmware update resource. Generated automatically by the system.",
			Description:         "ID of the IRMC firmware update resource.",
			Optional:            true,
			Computed:            true,
		},
		"update_type": schema.StringAttribute{
			MarkdownDescription: "Specifies the type of IRMC firmware update. Available options are: `File`, `TFTP`, and `MemoryCard`.",
			Description:         "Specifies the type of IRMC firmware update. Available options are: File, TFTP, and MemoryCard.",
			Required:            true,
			Validators: []validator.String{
				stringvalidator.OneOf([]string{
					UPDATE_TYPE_FILE,
					UPDATE_TYPE_TFTP,
					UPDATE_TYPE_MEMORY_CARD,
				}...),
			},
		},
		"irmc_path_to_binary": schema.StringAttribute{
			MarkdownDescription: "Path to the binary firmware file to upload when `update_type` is `File`. Accepted format: absolute file path.",
			Description:         "Path to the binary firmware file to upload when `update_type` is `File`. Accepted format: absolute file path.",
			Optional:            true,
			Computed:            true,
			Default:             stringdefault.StaticString(""),
			Validators: []validator.String{
				validators.ChangeToRequired(UPDATE_TYPE, UPDATE_TYPE_FILE),
			},
		},
		"tftp_server_addr": schema.StringAttribute{
			MarkdownDescription: "Address of the TFTP server when `update_type` is `TFTP`. Accepted format: valid IP address or hostname.",
			Description:         "Address of the TFTP server when `update_type` is `TFTP`. Accepted format: valid IP address or hostname.",
			Optional:            true,
			Computed:            true,
			Default:             stringdefault.StaticString(""),
			Validators: []validator.String{
				validators.ChangeToRequired(UPDATE_TYPE, UPDATE_TYPE_TFTP),
			},
		},
		"tftp_update_file": schema.StringAttribute{
			MarkdownDescription: "Path to the firmware file on the TFTP server when `update_type` is `TFTP`. Accepted format: relative file path (e.g., `/path/to/firmware.bin`).",
			Description:         "Path to the firmware file on the TFTP server when `update_type` is `TFTP`. Accepted format: relative file path (e.g., `/path/to/firmware.bin`).",
			Optional:            true,
			Computed:            true,
			Default:             stringdefault.StaticString(""),
			Validators: []validator.String{
				validators.ChangeToRequired(UPDATE_TYPE, UPDATE_TYPE_TFTP),
			},
		},
		"irmc_flash_selector": schema.StringAttribute{
			MarkdownDescription: "Flash selector for the update. Possible options are: `Auto`, `LowFWImage`, and `HighFWImage`. Default value: `Auto`.",
			Description:         "Flash selector for the update. Possible options are: `Auto`, `LowFWImage`, and `HighFWImage`. Default value: `Auto`.",
			Computed:            true,
			Optional:            true,
			Default:             stringdefault.StaticString("Auto"),
			Validators: []validator.String{
				stringvalidator.OneOf([]string{
					"Auto",
					"LowFWImage",
					"HighFWImage",
				}...),
			},
		},
		"irmc_boot_selector": schema.StringAttribute{
			MarkdownDescription: "Boot selector for the update. Possible options are: `Auto`, `LowFWImage`, `HighFWImage`, `OldestFW`, `MostRecentProgrammedFW`, and `LeastRecentProgrammedFW`. Default value: `Auto`.",
			Description:         "Boot selector for the update. Possible options are: `Auto`, `LowFWImage`, `HighFWImage`, `OldestFW`, `MostRecentProgrammedFW`, and `LeastRecentProgrammedFW`. Default value: `Auto`.",
			Computed:            true,
			Optional:            true,
			Default:             stringdefault.StaticString("Auto"),
			Validators: []validator.String{
				stringvalidator.OneOf([]string{
					"Auto",
					"LowFWImage",
					"HighFWImage",
					"OldestFW",
					"MostRecentProgrammedFW",
					"LeastRecentProgrammedFW",
				}...),
			},
		},
		"update_timeout": schema.Int64Attribute{
			MarkdownDescription: "Maximum duration (in seconds) to wait for the Firmware Update operation to finish before aborting. This does not include the time required for iRMC availability after the update. Default value: `3000` seconds.",
			Description:         "Maximum duration (in seconds) to wait for the Firmware Update operation to finish before aborting. This does not include the time required for iRMC availability after the update. Default value: `3000` seconds.",
			Computed:            true,
			Optional:            true,
			Default:             int64default.StaticInt64(FIRMWARE_UPDATE_TIMEOUT),
			PlanModifiers: []planmodifier.Int64{
				int64planmodifier.RequiresReplace(),
			},
		},
		"reset_irmc_after_update": schema.BoolAttribute{
			MarkdownDescription: "Automatically reboot iRMC after flashing if set to `true`. If `false`, the user must reboot iRMC manually to complete the firmware update process. Default value: `true`.",
			Description:         "Automatically reboot iRMC after flashing if set to `true`. If `false`, the user must reboot iRMC manually to complete the firmware update process. Default value: `true`.",
			Optional:            true,
			Computed:            true,
			Default:             booldefault.StaticBool(true),
		},
	}
}
func (r *IrmcFirmwareUpdateResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "This resource is used to update the IRMC firmware.",
		Description:         "This resource is used to update the IRMC firmware.",
		Attributes:          IrmcFirmwareUpdateSchema(),
		Blocks:              RedfishServerResourceBlockMap(),
	}
}

// Create handles the creation of the firmware update resource.
func (r *IrmcFirmwareUpdateResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	tflog.Info(ctx, "resource-irmc-redfish_irmc_firmware_update: create starts")

	// Get Plan Data
	var plan models.IrmcFirmwareUpdateResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Connect to the target system.
	api, err := ConnectTargetSystem(r.p, &plan.RedfishServer)
	if err != nil {
		resp.Diagnostics.AddError("Service Connection Error", err.Error())
		return
	}
	defer api.Logout()

	isFsas, err := IsFsasCheck(ctx, api)
	if err != nil {
		resp.Diagnostics.AddError("Vendor Detection Failed", err.Error())
		return
	}

	firmwareUpdEnpd := getFirmwareEndpoints(isFsas)

	err = setSelectors(api, &plan, firmwareUpdEnpd.FirmwareUpdateEndpoint)
	if err != nil {
		resp.Diagnostics.AddError("Failed to set iRMC Selectors", err.Error())
		return
	}

	// Handle firmware update based on the update type.
	switch plan.UpdateType.ValueString() {
	case UPDATE_TYPE_FILE:
		taskLocation, err := handleFileUpdate(api, &plan, firmwareUpdEnpd.FileFirmwareUpdateEndpoint)
		if err != nil {
			resp.Diagnostics.AddError("File firmware update failed.", err.Error())
			return
		}
		err = checkFirmwareUpdateStatus(ctx, api.Service, taskLocation, plan.UpdateTimeout.ValueInt64(), isFsas)
		if err != nil {
			resp.Diagnostics.AddError("File Firmware Update task did not complete successfully", err.Error())
			return
		}
	case UPDATE_TYPE_TFTP:
		taskLocation, err := handleTftpUpdate(api, &plan, firmwareUpdEnpd.FirmwareUpdateEndpoint, firmwareUpdEnpd.TftpFirmwareUpdateEndpoint)
		if err != nil {
			resp.Diagnostics.AddError("TFTP firmware update failed.", err.Error())
			return
		}
		err = checkFirmwareUpdateStatus(ctx, api.Service, taskLocation, plan.UpdateTimeout.ValueInt64(), isFsas)
		if err != nil {
			resp.Diagnostics.AddError("TFTP Firmware Update task did not complete successfully", err.Error())
			return
		}
	case UPDATE_TYPE_MEMORY_CARD:
		taskLocation, err := handleMemoryCardUpdate(api, firmwareUpdEnpd.MemoryCardFirmwareUpdateEndpoint)
		if err != nil {
			resp.Diagnostics.AddError("MemoryCard firmware update failed.", err.Error())
			return
		}
		err = checkFirmwareUpdateStatus(ctx, api.Service, taskLocation, plan.UpdateTimeout.ValueInt64(), isFsas)
		if err != nil {
			resp.Diagnostics.AddError("Memory Card Firmware Update task did not complete successfully", err.Error())
			return
		}
	}

	err = ResetIrmcAfterFirmwareUpd(ctx, api, &plan, r.p)
	if err != nil {
		resp.Diagnostics.AddError("Failed to reset iRMC after firmware update", err.Error())
		return
	}

	plan.Id = types.StringValue(firmwareUpdEnpd.FirmwareUpdateEndpoint)

	diags = resp.State.Set(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Info(ctx, "resource-irmc-redfish_irmc_firmware_update: create ends")
}

// Read handles reading the resource state.
func (r *IrmcFirmwareUpdateResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	tflog.Info(ctx, "resource-irmc-redfish_irmc_firmware_update: read starts")
	var state models.IrmcFirmwareUpdateResourceModel

	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Save into State
	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	tflog.Info(ctx, "resource-irmc-redfish_irmc_firmware_update: read ends")
}

// Update modifies the resource state but returns an error if triggered, as updates are not supported.
func (r *IrmcFirmwareUpdateResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// This function should not be called since updates are not supported; the resource should be recreated instead.
	resp.Diagnostics.AddError(
		"Unsupported Update Operation for IRMC Firmware Update",
		"The IRMC Firmware Update resource does not support in-place updates. It is intended to be destroyed and recreated if changes are required.",
	)
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *IrmcFirmwareUpdateResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	tflog.Info(ctx, "resource-irmc-redfish_irmc_firmware_update: delete starts")
	// Delete State Data
	resp.State.RemoveResource(ctx)
	tflog.Info(ctx, "resource-irmc-redfish_irmc_firmware_update: delete ends")
}

func handleTftpUpdate(api *gofish.APIClient, plan *models.IrmcFirmwareUpdateResourceModel, firmwareUpdateEndpoint, tftpFirmwareUpdateEndpoint string) (string, error) {

	res, err := api.Get(firmwareUpdateEndpoint)
	if err != nil {
		return "", fmt.Errorf("failed to fetch data from Redfish endpoint: %v", err)
	}
	defer res.Body.Close()

	payload := map[string]interface{}{
		"ServerName":   plan.TftpServerAddr.ValueString(),
		"iRMCFileName": plan.TftpUpdateFile.ValueString(),
	}

	res, err = api.PatchWithHeaders(firmwareUpdateEndpoint, payload,
		map[string]string{HTTP_HEADER_IF_MATCH: res.Header.Get(HTTP_HEADER_ETAG)})
	if err != nil {
		return "", fmt.Errorf("failed to send PATCH request: %v", err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK && res.StatusCode != http.StatusNoContent && res.StatusCode != http.StatusAccepted {
		return "", fmt.Errorf("PATCH request failed with status code: %d", res.StatusCode)
	}

	updatePayload := map[string]interface{}{}
	res, err = api.Post(tftpFirmwareUpdateEndpoint, updatePayload)
	if err != nil {
		return "", fmt.Errorf("failed to send POST request: %v", err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK && res.StatusCode != http.StatusNoContent && res.StatusCode != http.StatusAccepted {
		return "", fmt.Errorf("iRMC TFTP Firmware Update status code: %d", res.StatusCode)
	}

	taskLocation := res.Header.Get(HTTP_HEADER_LOCATION)
	if taskLocation == "" {
		return "", fmt.Errorf("task Location Missing. Location header not found in response")
	}
	return taskLocation, nil
}

func handleMemoryCardUpdate(api *gofish.APIClient, memoryCardFirmwareUpdateEndpoint string) (string, error) {

	payload := map[string]interface{}{}

	res, err := api.Post(memoryCardFirmwareUpdateEndpoint, payload)
	if err != nil {
		return "", fmt.Errorf("failed to send POST request: %v", err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK && res.StatusCode != http.StatusNoContent && res.StatusCode != http.StatusAccepted {
		return "", fmt.Errorf("iRMC MemoryCard Firmware Update status code: %d", res.StatusCode)
	}
	taskLocation := res.Header.Get(HTTP_HEADER_LOCATION)
	if taskLocation == "" {
		return "", fmt.Errorf("task Location Missing. Location header not found in response")
	}
	return taskLocation, nil
}

func handleFileUpdate(api *gofish.APIClient, plan *models.IrmcFirmwareUpdateResourceModel, fileFirmwareUpdateEndpoint string) (string, error) {
	if plan.IRMCPathToBinary.IsNull() {
		return "", fmt.Errorf("missing firmware file name in the configuration")
	}

	fileData, err := readFirmwareFile(plan.IRMCPathToBinary.ValueString())
	if err != nil {
		return "", fmt.Errorf("error reading firmware file: %w", err)
	}

	taskLocation, err := sendFileFirmwareUpdate(api, fileData, fileFirmwareUpdateEndpoint)
	if err != nil {
		return "", fmt.Errorf("error sending firmware update: %w", err)
	}

	return taskLocation, nil
}

func readFirmwareFile(filePath string) (*os.File, error) {

	if strings.ToLower(filepath.Ext(filePath)) != ".bin" {
		return nil, fmt.Errorf("invalid file type: %s, only .bin files are allowed", filepath.Ext(filePath))
	}

	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return nil, fmt.Errorf("firmware file not found at %s", filePath)
	}

	data, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("unable to read file %s: %s", filePath, err)
	}

	return data, nil
}

func sendFileFirmwareUpdate(api *gofish.APIClient, fileData *os.File, fileFirmwareUpdateEndpoint string) (string, error) {

	payload := map[string]io.Reader{
		"data": fileData,
	}

	resp, err := api.Service.GetClient().PostMultipart(fileFirmwareUpdateEndpoint, payload)
	if err != nil {
		return "", fmt.Errorf("error sending firmware update: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return "", fmt.Errorf("failed to read response body: %w", err)
		}
		return "", fmt.Errorf("failed to update firmware, status code: %d, response: %s", resp.StatusCode, string(body))
	}

	taskLocation := resp.Header.Get(HTTP_HEADER_LOCATION)
	if taskLocation == "" {
		return "", fmt.Errorf("task Location Missing. Location header not found in response")
	}

	return taskLocation, nil
}

func setSelectors(api *gofish.APIClient, plan *models.IrmcFirmwareUpdateResourceModel, firmwareUpdateEndpoint string) error {

	res, err := api.Get(firmwareUpdateEndpoint)
	if err != nil {
		return fmt.Errorf("failed to fetch data from Redfish endpoint: %w", err)
	}
	defer res.Body.Close()

	payload := map[string]interface{}{
		"iRMCBootSelector":  plan.IRMCBootSelector.ValueString(),
		"iRMCFlashSelector": plan.IRMCFlashSelector.ValueString(),
	}

	res, err = api.PatchWithHeaders(firmwareUpdateEndpoint, payload, map[string]string{
		HTTP_HEADER_IF_MATCH: res.Header.Get(HTTP_HEADER_ETAG),
	})
	if err != nil {
		return fmt.Errorf("failed to send PATCH request: %w", err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK && res.StatusCode != http.StatusNoContent && res.StatusCode != http.StatusAccepted {
		body, err := io.ReadAll(res.Body)
		if err != nil {
			return fmt.Errorf("failed to read response body: %w", err)
		}
		return fmt.Errorf("PATCH request failed with status code: %d, response: %s", res.StatusCode, string(body))
	}

	return nil
}

func checkFirmwareUpdateStatus(ctx context.Context, service *gofish.Service, location string, timeout int64, isFsas bool) error {
	finishedSuccessfully, err := WaitForRedfishTaskEnd(ctx, service, location, timeout)
	if err != nil || !finishedSuccessfully {
		taskLog, diags := FetchRedfishTaskLog(service, location, isFsas)
		if diags.HasError() {
			return fmt.Errorf("firmware Update task did not complete successfully: %s", err)
		}
		return fmt.Errorf("firmware Update task failed. Details: %s. Task log: %s", err, string(taskLog))
	}
	return nil
}

func ResetIrmcAfterFirmwareUpd(ctx context.Context, api *gofish.APIClient, plan *models.IrmcFirmwareUpdateResourceModel, provider *IrmcProvider) error {
	poweredOn, err := isPoweredOn(api.Service)
	if err != nil {
		return fmt.Errorf("failed to check power state: %w", err)
	}

	if poweredOn && plan.ResetIrmcAfterUpdate.ValueBool() {
		irmc, err := api.Service.Managers()
		if err != nil {
			return fmt.Errorf("error when accessing Managers resource: %w", err)
		}
		err = irmc[0].Reset(redfish.GracefulRestartResetType)
		if err != nil {
			return fmt.Errorf("error resetting manager: %w", err)
		}
	}

	api, err = retryConnectWithTimeout(ctx, provider, &plan.RedfishServer)
	if err != nil {
		return fmt.Errorf("service connect target system error: %w", err)
	}

	err = checkIrmcStatus(ctx, api, CHECK_INTERVAL, RESET_TIMEOUT)
	if err != nil {
		return fmt.Errorf("failed to reboot iRMC: %w", err)
	}

	return nil
}

func getFirmwareEndpoints(isFsas bool) firmwareUpdateEndpoints {
	if isFsas {
		return firmwareUpdateEndpoints{
			FirmwareUpdateEndpoint:           fmt.Sprintf("/redfish/v1/Managers/iRMC/Oem/%s/iRMCConfiguration/FWUpdate", FSAS),
			FileFirmwareUpdateEndpoint:       fmt.Sprintf("/redfish/v1/Managers/iRMC/Actions/Oem/%sManager.FWUpdate", FSAS),
			TftpFirmwareUpdateEndpoint:       fmt.Sprintf("/redfish/v1/Managers/iRMC/Actions/Oem/%sManager.FWTFTPUpdate", FSAS),
			MemoryCardFirmwareUpdateEndpoint: fmt.Sprintf("/redfish/v1/Managers/iRMC/Actions/Oem/%sManager.FWMemoryCardUpdate", FSAS),
		}
	} else {
		return firmwareUpdateEndpoints{
			FirmwareUpdateEndpoint:           fmt.Sprintf("/redfish/v1/Managers/iRMC/Oem/%s/iRMCConfiguration/FWUpdate", TS_FUJITSU),
			FileFirmwareUpdateEndpoint:       fmt.Sprintf("/redfish/v1/Managers/iRMC/Actions/Oem/%sManager.FWUpdate", FTS),
			TftpFirmwareUpdateEndpoint:       fmt.Sprintf("/redfish/v1/Managers/iRMC/Actions/Oem/%sManager.FWTFTPUpdate", FTS),
			MemoryCardFirmwareUpdateEndpoint: fmt.Sprintf("/redfish/v1/Managers/iRMC/Actions/Oem/%sManager.FWMemoryCardUpdate", FTS),
		}
	}

}
