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

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/stmcginnis/gofish"
	"github.com/stmcginnis/gofish/common"
	"github.com/stmcginnis/gofish/redfish"
)

type raidCapabilitiesConfig struct {
	RaidLevelCap []struct {
		RaidType                string   `json:"RAIDType"`
		StripeSizes             []int    `json:"StripeSizes"`
		StripeSizesHDD          []int    `json:"StripeSizesHDD"`
		StripeSizesSSD          []int    `json:"StripeSizesSSD"`
		StripeSizesNVMe         []int    `json:"StripeSizesNVMe"`
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

type volumeOem struct {
	Name           string `json:"Name,omitempty"`
	InitMode       string `json:"InitMode,omitempty"`
	ReadMode       string `json:"ReadMode,omitempty"`
	WriteMode      string `json:"WriteMode,omitempty"`
	DriveCacheMode string `json:"DriveCacheMode,omitempty"`
}

type volumeOemObject struct {
	OemFsas    *volumeOem `json:"Fsas,omitempty"`
	OemFujitsu *volumeOem `json:"ts_fujitsu,omitempty"`
}

type volumeObject struct {
	Oem volumeOemObject `json:"Oem"`
}

type physical_disk_group struct {
	Group []string
}

// getSystemStorageOemRaidCapabilitiesResource tries to access RAIDCapabilities endpoint
// related with RAID storage endpoint and returns response as structure in case of success.
func getSystemStorageOemRaidCapabilitiesResource(service *gofish.Service, endpoint string) (raidCapabilitiesConfig, error) {
	res, err := service.GetClient().Get(endpoint)
	var config raidCapabilitiesConfig
	if err != nil {
		return config, fmt.Errorf("could not access RAIDCapabilities resource due to: %s", err.Error())
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return config, fmt.Errorf("could not access RAIDCapabilities resource, http code %d", res.StatusCode)
	}

	bodyBytes, err := io.ReadAll(res.Body)
	if err != nil {
		return config, fmt.Errorf("error while reading response body: %s", err.Error())
	}

	err = json.Unmarshal(bodyBytes, &config)
	if err != nil {
		return config, fmt.Errorf("error during body unmarshalling: %s", err.Error())
	}

	return config, nil
}

func getVolumesCollectionUrl(service *gofish.Service, serial string) (url string, err error) {
	storage, err := getSystemStorageFromSerialNumber(service, serial)
	if err != nil {
		return "", fmt.Errorf("storage resource could not be obtained %s", err.Error())
	}

	return storage.ODataID + "/Volumes", err
}

// validateRequestAgainstStorageControllerCapabilities validates plan
// what target controller reports as supported. If validation has been successful,
// function returns slice of physical_disk_group.
func validateRequestAgainstStorageControllerCapabilities(ctx context.Context, service *gofish.Service,
	storage_id string, is_fsas bool, plan models.StorageVolumeResourceModel) ([]physical_disk_group, error) {
	physical_disk_groups := []physical_disk_group{}

	storage, err := getSystemStorageFromSerialNumber(service, storage_id)
	if err != nil {
		return physical_disk_groups, fmt.Errorf("storage resource could not be obtained %s", err.Error())
	}

	physical_disk_groups, drives_media_type, err := verifyRequestedDisks(ctx, plan, storage)
	if err != nil {
		return physical_disk_groups, fmt.Errorf("storage disk verification failed %s", err.Error())
	}

	// Obtain RAIDCapabilities for particular storage controller
	raidc_endpoint := storage.ODataID
	if is_fsas {
		raidc_endpoint = raidc_endpoint + STORAGE_RAIDCAPABILITIES_FSAS_SUFFIX
	} else {
		raidc_endpoint = raidc_endpoint + STORAGE_RAIDCAPABILITIES_SUFFIX
	}

	var capabilities raidCapabilitiesConfig
	capabilities, err = getSystemStorageOemRaidCapabilitiesResource(service, raidc_endpoint)
	if err != nil {
		return physical_disk_groups, fmt.Errorf("storage controller capabilities could not be obtained %s", err.Error())
	}

	// Validate request against what controller supports
	validated_raid_type := false
	validated_optimum_io_size_bytes := false

	for _, val := range capabilities.RaidLevelCap {
		if val.RaidType == plan.RaidType.ValueString() {
			validated_raid_type = true

			if len(val.StripeSizes) > 0 {
				for _, supp_iosize := range val.StripeSizes {
					if supp_iosize == int(plan.OptimumIOSizeBytes.ValueInt64()) {
						validated_optimum_io_size_bytes = true
						break
					}
				}
			} else {
				if drives_media_type == "SSD" {
					for _, supp_iosize := range val.StripeSizesSSD {
						if supp_iosize == int(plan.OptimumIOSizeBytes.ValueInt64()) {
							validated_optimum_io_size_bytes = true
							break
						}
					}
				} else if drives_media_type == "HDD" {
					for _, supp_iosize := range val.StripeSizesHDD {
						if supp_iosize == int(plan.OptimumIOSizeBytes.ValueInt64()) {
							validated_optimum_io_size_bytes = true
							break
						}
					}
				}
			}

			// Verify groups size
			num_of_groups := len(physical_disk_groups)
			if val.MinimumSpanCount != 0 && val.MaximumSpanCount != 0 {
				if num_of_groups < val.MinimumSpanCount || num_of_groups > val.MaximumSpanCount {
					return physical_disk_groups, fmt.Errorf("requested number of disk groups %d does not match %s",
						num_of_groups, val.RaidType)
				}

				min_num_of_disks_in_group := val.MinimumDriveCount / val.MinimumSpanCount
				for i, group := range physical_disk_groups {
					if len(group.Group) < min_num_of_disks_in_group {
						return physical_disk_groups, fmt.Errorf("minimal number of disks in group %d is not fulfilled", i)
					}
				}
			} else {
				if num_of_groups != 1 {
					return physical_disk_groups, fmt.Errorf("for %s only single group of disks is supported", val.RaidType)
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
func verifyRequestedDisks(ctx context.Context, plan models.StorageVolumeResourceModel, storage *redfish.Storage) ([]physical_disk_group, redfish.MediaType, error) {
	var plan_physical_disks []string
	var drives_media_type redfish.MediaType
	plan.PhysicalDrives.ElementsAs(ctx, &plan_physical_disks, true)

	tflog.Info(ctx, "Details of PhysicalDrives", map[string]interface{}{
		"Groups": plan_physical_disks,
	})

	physical_disks := []physical_disk_group{}

	drives, err := storage.Drives()
	if err != nil {
		return physical_disks, drives_media_type, fmt.Errorf("could not read drives from target system %s", err.Error())
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
			return physical_disks, drives_media_type, fmt.Errorf("could not unmarshal requested Drives '%s'", err.Error())
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
					_, err = fmt.Fscanf(drive_s, "[ %d : %d : %d : %d ]",
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
						drives_media_type = drive.MediaType
						break
					}
				} else {
					if strconv.Itoa(slot) == disk {
						disk_found = true
						drives_media_type = drive.MediaType
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

	return physical_disks, drives_media_type, nil
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

	if plan.ReadMode != nil {
		read_mode := plan.ReadMode.Requested.ValueString()
		if len(read_mode) > 0 {
			volume_config["ReadMode"] = read_mode
		}
	}

	if plan.WriteMode != nil {
		write_mode := plan.WriteMode.Requested.ValueString()
		if len(write_mode) > 0 {
			volume_config["WriteMode"] = write_mode
		}
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

// requestVolumeCreationAndSuperviseTheProcess sends creation request and waits until created task
// will finish.
func requestVolumeCreationAndSuperviseTheProcess(ctx context.Context, service *gofish.Service,
	volumes_collection_endpoint string, new_volume_payload map[string]interface{}, is_fsas bool, timeout int64) (diags diag.Diagnostics) {
	res, err := service.GetClient().Post(volumes_collection_endpoint, new_volume_payload)
	if err != nil {
		diags.AddError("Error while requesting POST on volume collection", err.Error())
		return diags
	}

	defer res.Body.Close()

	if res.StatusCode == http.StatusAccepted {
		task_location := res.Header.Get(HTTP_HEADER_LOCATION)
		_, err := WaitForRedfishTaskEnd(ctx, service, task_location, timeout)
		if err != nil {
			diags.AddError("Task for volume creation reported error", err.Error())
			logs, internal_diags := FetchRedfishTaskLog(service, task_location, is_fsas)
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

// requestAndSuperviseVolumeCreationProcess tries to create volume inside of service according to plan.
func requestAndSuperviseVolumeCreationProcess(ctx context.Context, api *gofish.APIClient,
	plan models.StorageVolumeResourceModel) (diags diag.Diagnostics) {

	storage_id := plan.StorageControllerSN.ValueString()

	is_fsas, err := IsFsasCheck(ctx, api)
	if err != nil {
		diags.AddError("Vendor detection failed", err.Error())
		return diags
	}

	physical_disk_groups, err := validateRequestAgainstStorageControllerCapabilities(ctx, api.Service, storage_id, is_fsas, plan)
	if err != nil {
		diags.AddError("Error during request validation", err.Error())
		return diags
	}

	new_volume_payload := getNewVolumeConfigFromPlan(plan, physical_disk_groups)

	volumes_collection_endpoint, err := getVolumesCollectionUrl(api.Service, storage_id)
	if err != nil {
		diags.AddError("Could not obtain volumes url", err.Error())
		return diags
	}

	tflog.Info(ctx, "Volume create request details", map[string]interface{}{
		"endpoint": volumes_collection_endpoint,
		"payload":  new_volume_payload,
	})

	return requestVolumeCreationAndSuperviseTheProcess(ctx, api.Service, volumes_collection_endpoint,
		new_volume_payload, is_fsas, plan.JobTimeout.ValueInt64())
}

// deleteStorageVolume tries to destroy volume_endpoint in service.
func deleteStorageVolume(ctx context.Context, service *gofish.Service,
	volume_endpoint string, is_fsas bool, timeout int64) diag.Diagnostics {

	var diags diag.Diagnostics

	res, err := service.GetClient().Delete(volume_endpoint)
	if err != nil {
		diags.AddError("Request to delete volume reported error", err.Error())
		return diags
	}

	defer res.Body.Close()

	if res.StatusCode == http.StatusAccepted {
		task_location := res.Header.Get(HTTP_HEADER_LOCATION)
		_, err := WaitForRedfishTaskEnd(ctx, service, task_location, timeout)
		if err != nil {
			diags.AddError("Task for volume deletion reported error", err.Error())
			logs, internal_diags := FetchRedfishTaskLog(service, task_location, is_fsas)
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

	if state.ReadMode != nil {
		if volumeOem.OemFsas != nil {
			state.ReadMode.Actual = types.StringValue(volumeOem.OemFsas.ReadMode)
		} else {
			state.ReadMode.Actual = types.StringValue(volumeOem.OemFujitsu.ReadMode)
		}
	}

	if state.WriteMode != nil {
		if volumeOem.OemFsas != nil {
			state.WriteMode.Actual = types.StringValue(volumeOem.OemFsas.WriteMode)
		} else {
			state.WriteMode.Actual = types.StringValue(volumeOem.OemFujitsu.WriteMode)
		}
	}

	if volumeOem.OemFsas != nil {
		state.DriveCacheMode = types.StringValue(volumeOem.OemFsas.DriveCacheMode)
	} else {
		state.DriveCacheMode = types.StringValue(volumeOem.OemFujitsu.DriveCacheMode)
	}

	return diags
}

// compareVolumePropertiesWithPlan reads current volume configuration and compare it in loop
// until planned changes will be reflected by volume configuration from service.
// The loop has timeout defined by timeout_s when operation will terminate if there will be still
// differences between plan and volume.
func compareVolumePropertiesWithPlan(ctx context.Context, service *gofish.Service, volume_id string,
	plan models.StorageVolumeResourceModel, timeout_s int64) (bool, error) {
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
			var driveCacheMode string
			if volumeOem.OemFujitsu != nil {
				driveCacheMode = volumeOem.OemFujitsu.DriveCacheMode
			} else {
				driveCacheMode = volumeOem.OemFsas.DriveCacheMode
			}

			if driveCacheMode == plan.DriveCacheMode.ValueString() {
				driveCacheVerified = true
			}
		}

		if nameVerified && driveCacheVerified {
			return true, nil
		}

		var driveCacheMode string
		if volumeOem.OemFujitsu != nil {
			driveCacheMode = volumeOem.OemFujitsu.DriveCacheMode
		} else {
			driveCacheMode = volumeOem.OemFsas.DriveCacheMode
		}

		tflog.Info(ctx, "compareVolumePropertiesWithPlan: compare plan with current volume",
			map[string]interface{}{
				"volume name (current)":      volume.Name,
				"volume name (planned)":      plan.VolumeName.ValueString(),
				"drive cache mode (current)": driveCacheMode,
				"drive cache mode (planned)": plan.DriveCacheMode.ValueString(),
			})

		if time.Now().Unix()-start_time > timeout_s {
			return false, fmt.Errorf("timeout of %d s has been reached", timeout_s)
		}

		time.Sleep(2 * time.Second)
	}
}

func waitUntilStorageVolumeChangesApplied(ctx context.Context, service *gofish.Service, taskLocation string, plan models.StorageVolumeResourceModel,
	volume_endpoint string, timeout int64) (status bool, err error) {

	if len(taskLocation) != 0 {
		return WaitForRedfishTaskEnd(ctx, service, taskLocation, timeout)
	}

	time.Sleep(5 * time.Second)

	// since no task is created, logic needs to wait with timeout for resource update
	return compareVolumePropertiesWithPlan(ctx, service, volume_endpoint, plan, timeout-5)
}

func patchVolumeEndpoint(ctx context.Context, service *gofish.Service, endpoint string, payload any) (taskLocation string, err error) {
	tflog.Info(ctx, "Volume change requested with payload", map[string]interface{}{
		"storage volume endpoint": endpoint,
		"payload":                 payload,
	})

	resp, err := service.GetClient().Patch(endpoint, payload)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
		return "", fmt.Errorf("PATCH request on '%s' finished with not expected status '%d'", endpoint, resp.StatusCode)
	}

	if resp.StatusCode == http.StatusAccepted {
		taskLocation := resp.Header.Get(HTTP_HEADER_LOCATION)
		if taskLocation == "" {
			return "", fmt.Errorf("Location header not found in response")
		}
		return taskLocation, nil
	} else {
		// Request might be accepted but some properties will not be successfully validated and it should be reported to terraform
		out, err := io.ReadAll(resp.Body)
		var respStruct StoragePatchResponse

		err = json.Unmarshal(out, &respStruct)
		if err != nil {
			return "", err
		}

		if len(respStruct.ExtendedInfo) > 0 {
			for _, v := range respStruct.ExtendedInfo {
				tflog.Warn(ctx, "Request responded with non-empty ExtendedMessageInfo", map[string]interface{}{
					"MessageId": v.MessageId,
					"Message":   v.Message,
				})
			}
		}

		return "", err
	}
}

// updateStorageVolume applies change on volume properties and verifies if planned
// changes are reflected by Redfish volume endpoint.
func requestVolumeModificationAndSuperviseTheProcess(ctx context.Context, service *gofish.Service, state models.StorageVolumeResourceModel,
	plan models.StorageVolumeResourceModel, is_fsas bool) (diags diag.Diagnostics) {

	var payload volumeObject
	var oem volumeOem

	if is_fsas {
		payload.Oem.OemFsas = &oem
	} else {
		payload.Oem.OemFujitsu = &oem
	}

	if !plan.DriveCacheMode.IsUnknown() {
		if payload.Oem.OemFsas != nil {
			payload.Oem.OemFsas.DriveCacheMode = plan.DriveCacheMode.ValueString()
		} else {
			payload.Oem.OemFujitsu.DriveCacheMode = plan.DriveCacheMode.ValueString()
		}
	}

	if !plan.VolumeName.IsUnknown() {
		if payload.Oem.OemFsas != nil {
			payload.Oem.OemFsas.Name = plan.VolumeName.ValueString()
		} else {
			payload.Oem.OemFujitsu.Name = plan.VolumeName.ValueString()
		}
	}

	volume_endpoint := state.Id.ValueString()

	task_location, err := patchVolumeEndpoint(ctx, service, volume_endpoint, payload)
	if err != nil {
		diags.AddError("Patch request to change volume parameters returned error", err.Error())
		return diags
	}

	_, err = waitUntilStorageVolumeChangesApplied(ctx, service, task_location, plan,
		volume_endpoint, plan.JobTimeout.ValueInt64())
	if err != nil {
		diags.AddError("Error while waiting for resource update.", err.Error())
		return diags
	}

	return diags
}

func updateStorageVolumeState(plan models.StorageVolumeResourceModel, target_volume_state models.StorageVolumeResourceModel,
	volume_endpoint string) models.StorageVolumeResourceModel {

	output := models.StorageVolumeResourceModel{
		Id:                  types.StringValue(volume_endpoint),
		StorageControllerSN: plan.StorageControllerSN,
		RedfishServer:       plan.RedfishServer,

		PhysicalDrives: plan.PhysicalDrives, // easier to be obtained from plan than from volume
		InitMode:       plan.InitMode,       // information not preserved in Redfish

		OptimumIOSizeBytes: target_volume_state.OptimumIOSizeBytes,
		RaidType:           target_volume_state.RaidType,
		VolumeName:         target_volume_state.VolumeName,
		CapacityBytes:      target_volume_state.CapacityBytes,
		DriveCacheMode:     target_volume_state.DriveCacheMode,
		JobTimeout:         target_volume_state.JobTimeout,
	}

	if plan.ReadMode != nil {
		output.ReadMode = &models.StorageVolumeDynamicParam{
			Requested: target_volume_state.ReadMode.Requested,
			Actual:    target_volume_state.ReadMode.Actual,
		}
	}

	if plan.WriteMode != nil {
		output.WriteMode = &models.StorageVolumeDynamicParam{
			Requested: target_volume_state.WriteMode.Requested,
			Actual:    target_volume_state.WriteMode.Actual,
		}
	}

	return output
}

func createStorageVolume(ctx context.Context, api *gofish.APIClient, plan models.StorageVolumeResourceModel, state *models.StorageVolumeResourceModel) (removeResource bool, diags diag.Diagnostics) {
	storage_id := plan.StorageControllerSN.ValueString()
	volumes_ids_before, diags := getVolumesIdsList(api.Service, storage_id)
	if diags.HasError() {
		return false, diags
	}

	diags = requestAndSuperviseVolumeCreationProcess(ctx, api, plan)
	if diags.HasError() {
		return false, diags
	}

	volumes_ids_after, diags := getVolumesIdsList(api.Service, storage_id)
	if diags.HasError() {
		return false, diags
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
		return true, diags
	}

	if volume == nil {
		if diags.HasError() {
			return false, diags
		}
	}

	if diags.HasError() {
		return false, diags
	}

	var target_volume_state models.StorageVolumeResourceModel
	target_volume_state.ReadMode = &models.StorageVolumeDynamicParam{}
	target_volume_state.WriteMode = &models.StorageVolumeDynamicParam{}
	if plan.ReadMode != nil {
		target_volume_state.ReadMode.Requested = plan.ReadMode.Requested
	}
	if plan.WriteMode != nil {
		target_volume_state.WriteMode.Requested = plan.WriteMode.Requested
	}

	diags = readStorageVolumeToState(volume, plan.StorageControllerSN.ValueString(),
		&target_volume_state)

	target_volume_state.JobTimeout = types.Int64Value(STORAGE_VOLUME_JOB_DEFAULT_TIMEOUT)
	if !plan.JobTimeout.IsUnknown() {
		target_volume_state.JobTimeout = plan.JobTimeout
	}

	localState := updateStorageVolumeState(plan, target_volume_state, new_volume_endpoint)
	*state = localState
	return false, diags
}

func updateStorageVolume(ctx context.Context, api *gofish.APIClient, plan models.StorageVolumeResourceModel, state *models.StorageVolumeResourceModel) (removeResource bool, diags diag.Diagnostics) {
	is_fsas, err := IsFsasCheck(ctx, api)
	if err != nil {
		diags.AddError("Vendor detection failed", err.Error())
		return false, diags
	}

	diags = requestVolumeModificationAndSuperviseTheProcess(ctx, api.Service, *state, plan, is_fsas)
	if diags.HasError() {
		return false, diags
	}

	tflog.Info(ctx, "resource-storage-volume: after update resource")

	volume, diags, beRemoved := doesVolumeStillExist(api.Service, state.Id.ValueString())
	if beRemoved {
		return true, diags
	}

	if volume == nil {
		if diags.HasError() {
			return false, diags
		}
	}

	diags = readStorageVolumeToState(volume, state.StorageControllerSN.ValueString(), state)
	if diags.HasError() {
		return false, diags
	}

	return false, diags
}
