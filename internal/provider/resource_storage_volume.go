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
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"terraform-provider-irmc-redfish/internal/models"

	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/stmcginnis/gofish"
	"github.com/stmcginnis/gofish/common"
	"github.com/stmcginnis/gofish/redfish"
)

var _ resource.Resource = &StorageVolumeResource{}
var _ resource.ResourceWithImportState = &StorageVolumeResource{}

func NewStorageVolumeResource() resource.Resource {
	return &StorageVolumeResource{}
}

// StorageVolumeResource defines the resource implementation.
type StorageVolumeResource struct {
	p *IrmcProvider
}

const (
	STORAGE_COLLECTION_ENDPOINT     = "/redfish/v1/Systems/0/Storage"
	STORAGE_RAIDCAPABILITIES_SUFFIX = "/Oem/ts_fujitsu/RAIDCapabilities"
	HTTP_HEADER_LOCATION            = "Location"
)

type raidCapabilitiesConfig struct {
	RaidLevelCap []struct {
		RaidType                string   `json:"RAIDType"`
		StripeSizes             []int    `json:"StripeSizes"`
		MinimumDriveCount       int      `json:"MinimumDriveCount"`
		MaximumDriveCount       int      `json:"MaximumDriveCount"`
		MinimumSpanCount        int      `json:"MinimumSpanCount"`
		MaximumSpanCount        int      `json:"MaximumSpanCount"`
		SupportedInitMode       []string `json:"SupportedInitMode"`
		SupportedReadMode       []string `json:"SupportedReadMode"`
		SupportedWriteMode      []string `json:"SupportedWriteMode"`
		SupportedDriveCacheMode []string `json:"SupportedDriveCacheMode"`
	} `json:"RAIDLevels"`
}

type tsVolumeObject struct {
	InitMode       string `json:"InitMode"`
	ReadMode       string `json:"ReadMode"`
	WriteMode      string `json:"WriteMode"`
	DriveCacheMode string `json:"DriveCacheMode"`
}

type volumeOemObject struct {
	Ts_fujitsu tsVolumeObject `json:"ts_fujitsu"`
}

type physical_disk_group struct {
	Group []string
}

func (r *StorageVolumeResource) updateStorageVolumeState(
	plan models.StorageVolumeResourceModel,
	target_volume_state models.StorageVolumeResourceModel,
	volume_endpoint string) models.StorageVolumeResourceModel {

	return models.StorageVolumeResourceModel{
		Id:                  types.StringValue(volume_endpoint),
		StorageControllerSN: plan.StorageControllerSN,
		RedfishServer:       plan.RedfishServer,

		PhysicalDrives: plan.PhysicalDrives, // easier to be obtained from plan than from volume
		InitMode:       plan.InitMode,       // information not preserved in Redfish

		OptimumIOSizeBytes: target_volume_state.OptimumIOSizeBytes,
		RaidType:           target_volume_state.RaidType,
		VolumeName:         target_volume_state.VolumeName,
		CapacityBytes:      target_volume_state.CapacityBytes,

		// Property marked as Computed are expected to return real values
		ReadMode:       target_volume_state.ReadMode,
		WriteMode:      target_volume_state.WriteMode,
		DriveCacheMode: target_volume_state.DriveCacheMode,
	}
}

// getSystemStorageFromSerialNumber returns pointer to storage resource
// represented by requested serial number.
func getSystemStorageFromSerialNumber(service *gofish.Service, serial string) (*redfish.Storage, error) {
	system, err := GetSystemResource(service)
	if err != nil {
		return nil, err
	}

	list_of_storage_controllers, err := system.Storage()
	if err != nil {
		return nil, err
	}

	for _, storage := range list_of_storage_controllers {
		if storage.StorageControllers[0].SerialNumber == serial {
			return storage, nil
		}
	}

	return nil, fmt.Errorf("Requested Storage resource has not been found on list")
}

// getSystemStorageOemRaidCapabilitiesResource tries to access RAIDCapabilities endpoint
// related with RAID storage endpoint and returns response as structure in case of success.
func getSystemStorageOemRaidCapabilitiesResource(service *gofish.Service, endpoint string) (raidCapabilitiesConfig, error) {
	res, err := service.GetClient().Get(endpoint)
	var config raidCapabilitiesConfig
	if err != nil {
		return config, fmt.Errorf("Could not access RAIDCapabilities resource due to: %s", err.Error())
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return config, fmt.Errorf("Could not access RAIDCapabilities resource, http code %d", res.StatusCode)
	}

	bodyBytes, err := io.ReadAll(res.Body)
	if err != nil {
		return config, fmt.Errorf("Error while reading response body: %s", err.Error())
	}

	err = json.Unmarshal(bodyBytes, &config)
	if err != nil {
		return config, fmt.Errorf("Error during body unmarshalling: %s", err.Error())
	}

	return config, nil
}

func getVolumesCollectionUrl(service *gofish.Service, serial string) (url string, err error) {
	storage, err := getSystemStorageFromSerialNumber(service, serial)
	if err != nil {
		return "", fmt.Errorf("Storage resource could not be obtained %s", err.Error())
	}

	return storage.ODataID + "/Volumes", err
}

// validateRequestAgainstStorageControllerCapabilities validates plan
// what target controller reports as supported. If validation has been successful,
// function returns slice of physical_disk_group.
func validateRequestAgainstStorageControllerCapabilities(ctx context.Context, service *gofish.Service,
	storage_id string, plan models.StorageVolumeResourceModel) ([]physical_disk_group, error) {
	physical_disk_groups := []physical_disk_group{}

	storage, err := getSystemStorageFromSerialNumber(service, storage_id)
	if err != nil {
		return physical_disk_groups, fmt.Errorf("Storage resource could not be obtained %s", err.Error())
	}

	physical_disk_groups, err = verifyRequestedDisks(ctx, plan, storage)
	if err != nil {
		return physical_disk_groups, fmt.Errorf("Storage disk verification failed %s", err.Error())
	}

	// Obtain RAIDCapabilities for particular storage controller
	raidc_endpoint := storage.ODataID + STORAGE_RAIDCAPABILITIES_SUFFIX
	var capabilities raidCapabilitiesConfig
	capabilities, err = getSystemStorageOemRaidCapabilitiesResource(service, raidc_endpoint)
	if err != nil {
		return physical_disk_groups, fmt.Errorf("Storage controller capabilities could not be obtained %s", err.Error())
	}

	// Validate request against what controller supports
	validated_raid_type := false
	validated_optimum_io_size_bytes := false

	for _, val := range capabilities.RaidLevelCap {
		if val.RaidType == plan.RaidType.ValueString() {
			validated_raid_type = true

			for _, supp_iosize := range val.StripeSizes {
				if supp_iosize == int(plan.OptimumIOSizeBytes.ValueInt64()) {
					validated_optimum_io_size_bytes = true
					break
				}
			}

			// Verify groups size
			num_of_groups := len(physical_disk_groups)
			if val.MinimumSpanCount != 0 && val.MaximumSpanCount != 0 {
				if num_of_groups < val.MinimumSpanCount || num_of_groups > val.MaximumSpanCount {
					return physical_disk_groups, fmt.Errorf("Requested number of disk groups %d does not match %s",
						num_of_groups, val.RaidType)
				}

				min_num_of_disks_in_group := val.MinimumDriveCount / val.MinimumSpanCount
				for i, group := range physical_disk_groups {
					if len(group.Group) < min_num_of_disks_in_group {
						return physical_disk_groups, fmt.Errorf("Minimal number of disks in group %d is not fulfilled", i)
					}
				}
			} else {
				if num_of_groups != 1 {
					return physical_disk_groups, fmt.Errorf("For %s only single group of disks is supported", val.RaidType)
				}
			}

			break
		}
	}

	if !validated_raid_type {
		return physical_disk_groups, fmt.Errorf("raid_type has not been successfully validated against controller possibilities '%v'", capabilities.RaidLevelCap)
	}

	if !validated_optimum_io_size_bytes {
		return physical_disk_groups, fmt.Errorf("optimum_io_size_bytes has not been successfully validated against controller possibilities '%v'", capabilities.RaidLevelCap)
	}

	if !plan.CapacityBytes.IsUnknown() {
		if strings.Contains(storage.Name, "PDUAL CP100") {
			return physical_disk_groups, fmt.Errorf("PDUAL CP100 controller supports only full volumes (capacity_bytes cannot be specified)")
		}
	}

	return physical_disk_groups, nil
}

// verifyRequestedDisks verifies requested plan around disks vs disks attached to
// requested storage controller and returns slice of physical_disk_group if all disks
// have been found on target.
func verifyRequestedDisks(ctx context.Context, plan models.StorageVolumeResourceModel, storage *redfish.Storage) ([]physical_disk_group, error) {
	var plan_physical_disks []string
	plan.PhysicalDrives.ElementsAs(ctx, &plan_physical_disks, true)

	tflog.Info(ctx, "Details of PhysicalDrives", map[string]interface{}{
		"Groups": plan_physical_disks,
	})

	physical_disks := []physical_disk_group{}

	drives, err := storage.Drives()
	if err != nil {
		return physical_disks, fmt.Errorf("Could not read drives from target system %s", err.Error())
	}

	for _, group := range plan_physical_disks {
		tflog.Info(ctx, "Details of a particular group", map[string]interface{}{
			"group": group,
		})

		// Every group of disks slots is string and must be converted
		// to slice of strings (slots)
		var disks_in_group []string
		err = json.Unmarshal([]byte(group), &disks_in_group)
		if err != nil {
			return physical_disks, fmt.Errorf("Could not unmarshal requested Drives '%s'", err.Error())
		}

		for _, disk := range disks_in_group {

			var disk_found bool = false
			for _, drive := range drives {
				if len(drive.Location) == 0 {
					continue
				}

				tflog.Info(ctx, "Disks location", map[string]interface{}{
					"Drive location": drive.Location[0].Info,
				})

				drive_s := strings.NewReader(drive.Location[0].Info)
				var (
					system     int
					controller int
					enclosure  int
					slot       int
				)

				// Differentiate between drives in enclosure and directly attached
				var err error
				enclosure_attached := false
				if drive.Location[0].InfoFormat == "[ System_Id : Controller_Id : Enclosure_Id : Slot_Id ]" {
					_, err = fmt.Fscanf(drive_s, "[ %d : %d : %d : %d]",
						&system, &controller, &enclosure, &slot)
					enclosure_attached = true
				} else {
					_, err = fmt.Fscanf(drive_s, "[ %d : %d : %d ]", &system, &controller, &slot)
				}

				if err != nil {
					tflog.Warn(ctx, "Scanning disk location failed", map[string]interface{}{
						"drive": drive_s,
					})
				}

				if enclosure_attached {
					if fmt.Sprintf("%d-%d", enclosure, slot) == disk {
						disk_found = true
						break
					}
				} else {
					if strconv.Itoa(slot) == disk {
						disk_found = true
						break
					}
				}
			}

			if !disk_found {
				tflog.Warn(ctx, "Disk slot has not been found on target system", map[string]interface{}{
					"requested disk": disk,
				})

				// Really not sure whether the logic will be able to successfully
				// validate all cases, so just raise a warning for now
				// return physical_disks
			}
		}

		physical_disks = append(physical_disks, physical_disk_group{Group: disks_in_group})
	}

	return physical_disks, nil
}

// getNewVolumeConfigFromPlan based on plan and already converted list of disks in physical_disks
// returns map containing whole request as map.
func getNewVolumeConfigFromPlan(plan models.StorageVolumeResourceModel,
	physical_disks []physical_disk_group) map[string]interface{} {

	volume_config := map[string]interface{}{
		"Name":          plan.VolumeName.ValueString(),
		"RAIDType":      plan.RaidType.ValueString(),
		"PhysicalDisks": physical_disks,
	}

	// Handle optional arguments if not provided by user, do not add them to request
	// as it might make more problems than benefits (some controllers will accept value but return null
	// or empty string in resource returned as response)
	capacity := plan.CapacityBytes.ValueInt64()
	if capacity != 0 {
		volume_config["CapacityBytes"] = capacity
	}

	init_mode := plan.InitMode.ValueString()
	if len(init_mode) > 0 {
		volume_config["InitMode"] = init_mode
	}

	read_mode := plan.ReadMode.ValueString()
	if len(read_mode) > 0 {
		volume_config["ReadMode"] = read_mode
	}

	write_mode := plan.WriteMode.ValueString()
	if len(write_mode) > 0 {
		volume_config["WriteMode"] = write_mode
	}

	drive_cache_mode := plan.DriveCacheMode.ValueString()
	if len(drive_cache_mode) > 0 {
		volume_config["DriveCacheMode"] = drive_cache_mode
	}

	stripe_size := plan.OptimumIOSizeBytes.ValueInt64()
	if stripe_size != 0 {
		volume_config["OptimumIOSizeBytes"] = stripe_size
	}

	return volume_config
}

// getVolumesIdsList access requested storage_id and returns slice of available volumes
// by their @odata.id.
func getVolumesIdsList(service *gofish.Service, storage_id string) (out []string, diags diag.Diagnostics) {
	storage, err := getSystemStorageFromSerialNumber(service, storage_id)
	if err != nil {
		diags.AddError("Could not obtain storage controller with requested id", err.Error())
		return
	}

	volumes, err := storage.Volumes()
	if err != nil {
		diags.AddError("Could not obtain volumes of storage controller with requested id", err.Error())
		return
	}

	for _, volume := range volumes {
		out = append(out, volume.ODataID)
	}
	return out, diags
}

// getRecentlyCreatedVolumeId compares two slices of volumes and returned the one
// which is new.
func getRecentlyCreatedVolumeId(ids_after, ids_before []string) string {
	diff := difference(ids_after, ids_before)
	if len(diff) > 0 {
		return diff[0]
	}

	return ""
}

// requestVolumeCreationAndSuperviseCreation sends creation request and waits until created task
// will finish.
func requestVolumeCreationAndSuperviseCreation(ctx context.Context, service *gofish.Service,
	volumes_collection_endpoint string, new_volume_payload map[string]interface{}) (diags diag.Diagnostics) {
	res, err := service.GetClient().Post(volumes_collection_endpoint, new_volume_payload)
	if err != nil {
		diags.AddError("Error while requesting POST on volume collection", err.Error())
		return diags
	}

	defer res.Body.Close()

	if res.StatusCode == http.StatusAccepted {
		task_location := res.Header.Get(HTTP_HEADER_LOCATION)
		_, err := WaitForRedfishTaskEnd(ctx, service, task_location, 300)
		if err != nil {
			diags.AddError("Task for volume creation reported error", err.Error())
			logs, internal_diags := FetchRedfishTaskLog(service, task_location)
			if logs == nil {
				diags = append(diags, internal_diags...)
			} else {
				diags.AddError("Task logs for volume creation", string(logs))
			}
		}

	} else {
		diags.AddError("POST request on volume collection finished with error", "Non 200")
	}
	return diags
}

// getValidStorageEndpointFromSerial returns storage which represents itself
// with requested serial number.
func getValidStorageEndpointFromSerial(service *gofish.Service, storage_serial string) (endpoint string, err error) {
	storage, err := getSystemStorageFromSerialNumber(service, storage_serial)
	if err != nil {
		return "", err
	}

	return storage.ODataID, err
}

// updateVolumeODataId checks if previously used endpoint still points to same
// controller. If not, it produces valid endpoint to same volume.
func updateVolumeODataId(validStorageEndpoint string, state *models.StorageVolumeResourceModel) bool {
	knownVolumeId := state.Id.ValueString()
	if strings.Contains(knownVolumeId, validStorageEndpoint) {
		return false
	}

	newODataId := validStorageEndpoint + "/Volumes/" + getStorageIdFromVolumeODataId(knownVolumeId)
	state.Id = types.StringValue(newODataId)

	return true
}

// createStorageVolume tries to create volume inside of service according to plan.
func createStorageVolume(ctx context.Context, service *gofish.Service,
	plan models.StorageVolumeResourceModel) (diags diag.Diagnostics) {

	storage_id := plan.StorageControllerSN.ValueString()

	physical_disk_groups, err := validateRequestAgainstStorageControllerCapabilities(ctx, service, storage_id, plan)
	if err != nil {
		diags.AddError("Error during request validation", err.Error())
		return diags
	}

	new_volume_payload := getNewVolumeConfigFromPlan(plan, physical_disk_groups)

	volumes_collection_endpoint, err := getVolumesCollectionUrl(service, storage_id)
	if err != nil {
		diags.AddError("Could not obtain volumes url", err.Error())
		return diags
	}

	tflog.Info(ctx, "Volume create request details", map[string]interface{}{
		"endpoint": volumes_collection_endpoint,
		"payload":  new_volume_payload,
	})

	return requestVolumeCreationAndSuperviseCreation(ctx, service, volumes_collection_endpoint, new_volume_payload)
}

// deleteStorageVolume tries to destroy volume_endpoint in service.
func deleteStorageVolume(ctx context.Context, service *gofish.Service,
	volume_endpoint string) diag.Diagnostics {
	var diags diag.Diagnostics

	res, err := service.GetClient().Delete(volume_endpoint)
	if err != nil {
		diags.AddError("Request to delete volume reported error", err.Error())
		return diags
	}

	defer res.Body.Close()

	if res.StatusCode == http.StatusAccepted {
		task_location := res.Header.Get(HTTP_HEADER_LOCATION)
		_, err := WaitForRedfishTaskEnd(ctx, service, task_location, 300)
		if err != nil {
			diags.AddError("Task for volume deletion reported error", err.Error())
			logs, internal_diags := FetchRedfishTaskLog(service, task_location)
			if logs == nil {
				diags = append(diags, internal_diags...)
			} else {
				diags.AddError("Task logs for volume creation", string(logs))
			}
		}
	} else {
		diags.AddError("DELETE request on volume collection finished with error", "Non 200")
	}

	return diags
}

// doesVolumeStillExist verifies if volume_endpoint still exist in service target.
// If volume exist, function returns the volume pointer, if it does not exist it provides
// information outside to clean up terraform resource.
func doesVolumeStillExist(service *gofish.Service, volume_endpoint string) (volume *redfish.Volume, diags diag.Diagnostics, remove bool) {
	volume, err := redfish.GetVolume(service.GetClient(), volume_endpoint)
	if err != nil {
		var err_detailed *common.Error
		if !errors.As(err, &err_detailed) {
			diags.AddError("Error with getting volume", err.Error())
			return nil, diags, false
		}

		if err_detailed.HTTPReturnedStatusCode == http.StatusNotFound {
			diags.AddError("Requested volume does not exist", volume_endpoint)
			return nil, diags, true
		} else {
			diags.AddError("Reading volume details failed", volume_endpoint)
			return nil, diags, false
		}
	}
	return volume, diags, false
}

// getStorageIdFromVolumeODataId tries to read storage id out of volumeOdataId.
func getStorageIdFromVolumeODataId(volumeOdataId string) string {
	suffix := strings.Index(volumeOdataId, "/Volume")
	output := volumeOdataId[:suffix]

	prefix := strings.LastIndex(output, "/")
	output = output[prefix+1:]

	return output
}

// readStorageVolumeToState reads current volume configuration to terraform state.
func readStorageVolumeToState(volume *redfish.Volume, storage_serial string,
	state *models.StorageVolumeResourceModel) (diags diag.Diagnostics) {

	state.StorageControllerSN = types.StringValue(storage_serial)
	state.VolumeName = types.StringValue(volume.Name)
	state.OptimumIOSizeBytes = types.Int64Value(int64(volume.OptimumIOSizeBytes))

	state.CapacityBytes = models.CapacityByteValue{Int64Value: types.Int64Value(int64(volume.CapacityBytes))}

	// Theoretically volume can be migrated to different RAID type
	state.RaidType = types.StringValue(string(volume.RAIDType))

	var volumeOem volumeOemObject
	err := json.Unmarshal(volume.OEM, &volumeOem)
	if err != nil {
		diags.AddError("Could not unmarshal volume resource OEM object", err.Error())
		return diags
	}

	state.ReadMode = types.StringValue(volumeOem.Ts_fujitsu.ReadMode)
	state.WriteMode = types.StringValue(volumeOem.Ts_fujitsu.WriteMode)
	state.DriveCacheMode = types.StringValue(volumeOem.Ts_fujitsu.DriveCacheMode)

	return diags
}

// compareVolumePropertiesWithPlan reads current volume configuration and compare it in loop
// until planned changes will be reflected by volume configuration from service.
// The loop has timeout defined by timeout_s when operation will terminate if there will be still
// differences between plan and volume.
func compareVolumePropertiesWithPlan(ctx context.Context, service *gofish.Service, volume_id string,
	plan *models.StorageVolumeResourceModel, timeout_s int64) (bool, error) {
	start_time := time.Now().Unix()

	nameVerified := true
	verifyVolumeName := false
	if !plan.VolumeName.IsUnknown() {
		verifyVolumeName = true
		nameVerified = false
	}

	driveCacheVerified := true
	verifyDriveCacheMode := false
	if !plan.DriveCacheMode.IsUnknown() {
		verifyDriveCacheMode = true
		driveCacheVerified = false
	}

	for {
		volume, err := redfish.GetVolume(service.GetClient(), volume_id)
		if err != nil {
			return false, err
		}

		var volumeOem volumeOemObject
		err = json.Unmarshal(volume.OEM, &volumeOem)
		if err != nil {
			return false, err
		}

		if verifyVolumeName {
			if volume.Name == plan.VolumeName.ValueString() {
				nameVerified = true
			}
		}

		if verifyDriveCacheMode {
			if volumeOem.Ts_fujitsu.DriveCacheMode == plan.DriveCacheMode.ValueString() {
				driveCacheVerified = true
			}
		}

		if nameVerified && driveCacheVerified {
			return true, nil
		}

		tflog.Info(ctx, "compareVolumePropertiesWithPlan: compare plan with current volume",
			map[string]interface{}{
				"volume name (current)":      volume.Name,
				"volume name (planned)":      plan.VolumeName.ValueString(),
				"drive cache mode (current)": volumeOem.Ts_fujitsu.DriveCacheMode,
				"drive cache mode (planned)": plan.DriveCacheMode.ValueString(),
			})

		if time.Now().Unix()-start_time > timeout_s {
			return false, fmt.Errorf("Timeout of %d s has been reached", timeout_s)
		}

		time.Sleep(2 * time.Second)
	}
}

// updateStorageVolume applies change on volume properties and verifies if planned
// changes are reflected by Redfish volume endpoint.
func updateStorageVolume(ctx context.Context, service *gofish.Service, state *models.StorageVolumeResourceModel,
	plan *models.StorageVolumeResourceModel) (diags diag.Diagnostics) {
	payload := map[string]map[string]map[string]string{
		"Oem": {
			"ts_fujitsu": {},
		},
	}

	if !plan.DriveCacheMode.IsUnknown() {
		payload["Oem"]["ts_fujitsu"]["DriveCacheMode"] = plan.DriveCacheMode.ValueString()
	}

	if !plan.VolumeName.IsUnknown() {
		payload["Oem"]["ts_fujitsu"]["Name"] = plan.VolumeName.ValueString()
	}

	tflog.Info(ctx, "Volume change requested with payload", map[string]interface{}{
		"requested_payload": fmt.Sprintf("%v", payload),
	})

	volume_endpoint := state.Id.ValueString()
	res, err := service.GetClient().Patch(volume_endpoint, payload)

	if err != nil {
		diags.AddError("Patch request to change volume parameters returned error", err.Error())
		return diags
	}

	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		diags.AddError("Request to change volume parameters finished with error", "")
		return diags
	} else {
		time.Sleep(5 * time.Second)

		// since no task is created, logic needs to wait with timeout for resource update
		_, err := compareVolumePropertiesWithPlan(ctx, service, volume_endpoint, plan, 60)
		if err != nil {
			diags.AddError("Failed to change parameters", err.Error())
		}
	}

	return diags
}

func (r *StorageVolumeResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + storageVolumeName
}

func StorageVolumeSchema() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"id": schema.StringAttribute{
			Computed:            true,
			Description:         "Id of handled volume",
			MarkdownDescription: "Id of handled volume",
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.UseStateForUnknown(),
			},
		},
		"storage_controller_serial_number": schema.StringAttribute{
			Required:            true,
			Description:         "Serial number of storage controller.",
			MarkdownDescription: "Serial number of storage controller.",
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.RequiresReplace(),
			},
		},
		"raid_type": schema.StringAttribute{
			Required:            true,
			Description:         "RAID volume type depending on controller itself",
			MarkdownDescription: "RAID volume type depending on controller itself",
			Validators: []validator.String{
				stringvalidator.OneOf([]string{
					"RAID0",
					"RAID1",
					"RAID1E",
					"RAID10",
					"RAID5",
					"RAID50",
					"RAID6",
					"RAID60",
				}...),
			},
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.RequiresReplace(),
			},
		},
		"physical_drives": schema.ListAttribute{
			Required:            true,
			Description:         "List of slot locations of disks used for volume creation.",
			MarkdownDescription: "List of slot locations of disks used for volume creation.",
			ElementType:         types.StringType,
			Validators: []validator.List{
				listvalidator.SizeAtLeast(1),
			},
			PlanModifiers: []planmodifier.List{
				listplanmodifier.RequiresReplace(),
			},
		},
		// Usually if volume is created, size of the volume is not exactly
		// the same as requested due to controller (values in bytes can be rounded).
		// For that reason semantic equality logic is required here.
		"capacity_bytes": schema.Int64Attribute{
			CustomType:          models.CapacityByteType{},
			Description:         "Volume capacity in bytes.",
			MarkdownDescription: "Volume capacity in bytes. If not specified during creation, volume will have maximum size calculated from chosen disks.",
			Optional:            true,
			Computed:            true,
			PlanModifiers: []planmodifier.Int64{
				int64planmodifier.RequiresReplaceIfConfigured(),
			},
		},
		"name": schema.StringAttribute{
			Optional:            true,
			Description:         "Volume name.",
			MarkdownDescription: "Volume name.",
			Validators: []validator.String{
				stringvalidator.LengthAtLeast(1),
				stringvalidator.LengthAtMost(15),
			},
		},
		"init_mode": schema.StringAttribute{
			Optional:            true,
			Description:         "Initialize mode for new volume.",
			MarkdownDescription: "Initialize mode for new volume.",
			Validators: []validator.String{
				stringvalidator.OneOf([]string{
					"None",
					"Fast",
					"Normal",
				}...),
			},
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.RequiresReplaceIfConfigured(),
			},
		},
		"optimum_io_size_bytes": schema.Int64Attribute{
			Description:         "Optimum IO size bytes.",
			MarkdownDescription: "Optimum IO size bytes.",
			Required:            true,
			PlanModifiers: []planmodifier.Int64{
				int64planmodifier.RequiresReplace(),
			},
		},
		"read_mode": schema.StringAttribute{
			Optional:            true,
			Description:         "Read mode of volume.",
			MarkdownDescription: "Read mode of volume.",
			Validators: []validator.String{
				stringvalidator.OneOf([]string{
					"Adaptive",
					"NoReadAhead",
					"ReadAhead",
				}...),
			},
			Computed: true,
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.RequiresReplaceIfConfigured(),
			},
		},
		"write_mode": schema.StringAttribute{
			Optional:            true,
			Description:         "Write mode of volume.",
			MarkdownDescription: "Write mode of volume.",
			Validators: []validator.String{
				stringvalidator.OneOf([]string{
					"WriteBack",
					"AlwaysWriteBack",
					"WriteThrough",
				}...),
			},
			Computed: true,
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.RequiresReplaceIfConfigured(),
			},
		},
		"drive_cache_mode": schema.StringAttribute{
			Optional:            true,
			Description:         "Drive cache mode of volume.",
			MarkdownDescription: "Drive cache mode of volume.",
			Validators: []validator.String{
				stringvalidator.OneOf([]string{
					"Enabled",
					"Disabled",
					"Unchanged",
				}...),
			},
			Computed: true,
		},
	}
}

func (r *StorageVolumeResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description:         "This resource is used to manipulate (Create, Read, Delete, Update and Import) logical volumes of iRMC system",
		MarkdownDescription: "This resource is used to manipulate (Create, Read, Delete, Update and Import) logical volumes of iRMC system",
		Attributes:          StorageVolumeSchema(),
		Blocks:              RedfishServerResourceBlockMap(),
	}
}

func (r *StorageVolumeResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *StorageVolumeResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	tflog.Info(ctx, "storage-volume: create starts")

	// Read Terraform plan data into the model
	var plan models.StorageVolumeResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Provide synchronization
	var endpoint string = plan.RedfishServer[0].Endpoint.ValueString()
	var resource_name string = "resource-storage_volume"
	mutexPool.Lock(ctx, endpoint, resource_name)
	defer mutexPool.Unlock(ctx, endpoint, resource_name)

	api, err := ConnectTargetSystem(r.p, &plan.RedfishServer)
	if err != nil {
		resp.Diagnostics.AddError("Connection to service failed: ", err.Error())
		return
	}

	defer api.Logout()

	storage_id := plan.StorageControllerSN.ValueString()
	volumes_ids_before, diags := getVolumesIdsList(api.Service, storage_id)
	resp.Diagnostics.Append(diags...)
	if diags.HasError() {
		return
	}

	diags = createStorageVolume(ctx, api.Service, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	volumes_ids_after, diags := getVolumesIdsList(api.Service, storage_id)
	resp.Diagnostics.Append(diags...)
	if diags.HasError() {
		return
	}

	new_volume_endpoint := getRecentlyCreatedVolumeId(
		volumes_ids_after, volumes_ids_before)

	tflog.Trace(ctx, "Information about volume request", map[string]interface{}{
		"before": volumes_ids_before,
		"after":  volumes_ids_after,
		"new":    new_volume_endpoint,
	})

	// Update state based on created volume details
	volume, diags, to_remove := doesVolumeStillExist(api.Service, new_volume_endpoint)
	if to_remove {
		resp.State.RemoveResource(ctx)
		return
	}

	if volume == nil {
		resp.Diagnostics.Append(diags...)
		if diags.HasError() {
			return
		}
	}

	resp.Diagnostics.Append(diags...)
	if diags.HasError() {
		return
	}

	var target_volume_state models.StorageVolumeResourceModel
	diags = readStorageVolumeToState(volume, plan.StorageControllerSN.ValueString(),
		&target_volume_state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	state := r.updateStorageVolumeState(plan, target_volume_state, new_volume_endpoint)
	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)

	tflog.Info(ctx, "resource-storage-volume: create ends")
}

func (r *StorageVolumeResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	tflog.Info(ctx, "resource-storage-volume: read starts")

	// Read Terraform prior state data into the model
	var state models.StorageVolumeResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Connect to service
	api, err := ConnectTargetSystem(r.p, &state.RedfishServer)
	if err != nil {
		resp.Diagnostics.AddError("Connection to service failed: ", err.Error())
		return
	}

	defer api.Logout()

	validStorageEndpoint, err := getValidStorageEndpointFromSerial(api.Service, state.StorageControllerSN.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to get valid storage id", err.Error())
		return
	}

	if updateVolumeODataId(validStorageEndpoint, &state) {
		tflog.Info(ctx, "resource-storage-volume: state controller changed its Id")
	} else {
		tflog.Info(ctx, "resource-storage-volume: storage controller has stable id")
	}

	volume, diags, to_remove := doesVolumeStillExist(api.Service, state.Id.ValueString())
	if to_remove {
		resp.State.RemoveResource(ctx)
		return
	}

	if volume == nil {
		resp.Diagnostics.Append(diags...)
		if diags.HasError() {
			return
		}
	}

	resp.Diagnostics.Append(diags...)
	if diags.HasError() {
		return
	}

	diags = readStorageVolumeToState(volume, state.StorageControllerSN.ValueString(), &state)
	resp.Diagnostics.Append(diags...)

	if diags.HasError() {
		return
	}

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)

	tflog.Info(ctx, "resource-storage-volume: read ends")
}

func (r *StorageVolumeResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	tflog.Info(ctx, "resource-storage-volume: update starts")

	// Read Terraform state and plan data into the model
	var state models.StorageVolumeResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var plan models.StorageVolumeResourceModel
	diags = req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Connect to service
	api, err := ConnectTargetSystem(r.p, &plan.RedfishServer)
	if err != nil {
		resp.Diagnostics.AddError("Connection to service failed: ", err.Error())
		return
	}

	defer api.Logout()

	diags = updateStorageVolume(ctx, api.Service, &state, &plan)
	tflog.Info(ctx, "resource-storage-volume: after update resource")
	resp.Diagnostics.Append(diags...)

	if diags.HasError() {
		return
	}

	volume, diags, to_remove := doesVolumeStillExist(api.Service, state.Id.ValueString())
	if to_remove {
		resp.State.RemoveResource(ctx)
		return
	}

	if volume == nil {
		resp.Diagnostics.Append(diags...)
		if diags.HasError() {
			return
		}
	}

	resp.Diagnostics.Append(diags...)
	if diags.HasError() {
		return
	}

	diags = readStorageVolumeToState(volume, state.StorageControllerSN.ValueString(), &state)
	resp.Diagnostics.Append(diags...)
	if diags.HasError() {
		return
	}

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)

	tflog.Info(ctx, "resource-storage-volume: update ends")
}

func (r *StorageVolumeResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	tflog.Info(ctx, "resource-storage-volume: delete starts")

	// Read Terraform prior state data into the model
	var state models.StorageVolumeResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Connect to service
	api, err := ConnectTargetSystem(r.p, &state.RedfishServer)
	if err != nil {
		resp.Diagnostics.AddError("Connection to service failed: ", err.Error())
		return
	}

	defer api.Logout()

	// Try to delete handled volume
	diags = deleteStorageVolume(ctx, api.Service, state.Id.ValueString())
	resp.Diagnostics.Append(diags...)

	if diags.HasError() {
		return
	}

	tflog.Info(ctx, "resource-storage-volume: delete ends")
}

type StorageVolumeImportConfig struct {
	ServerConfig
	ID string `json:"id"`
}

func (r *StorageVolumeResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	tflog.Info(ctx, "resource-storage-volume: import starts")

	var config StorageVolumeImportConfig
	err := json.Unmarshal([]byte(req.ID), &config)
	if err != nil {
		resp.Diagnostics.AddError("Could not import configuration", err.Error())
		return
	}

	server := models.RedfishServer{
		User:        types.StringValue(config.Username),
		Password:    types.StringValue(config.Password),
		Endpoint:    types.StringValue(config.Endpoint),
		SslInsecure: types.BoolValue(config.SslInsecure),
	}

	// no need to read current configuration since terraform will call Read() once
	// import procedure will be successfully finished

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), config.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("server"), []models.RedfishServer{server})...)

	tflog.Info(ctx, "resource-storage-volume: import ends")
}
