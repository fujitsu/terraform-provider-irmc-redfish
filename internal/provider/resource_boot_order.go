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
	"strings"
	"time"

	"terraform-provider-irmc-redfish/internal/models"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
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
var _ resource.Resource = &BootOrderResource{}
var _ resource.ResourceWithImportState = &BootOrderResource{}

func NewBootOrderResource() resource.Resource {
	return &BootOrderResource{}
}

// BootOrderResource defines the resource implementation.
type BootOrderResource struct {
	p *IrmcProvider
}

type BootOrder []string
type BootOrderEntry struct {
	DeviceName           string
	StructuredBootString string
}

type BiosSettings struct {
	Attributes redfish.SettingsAttributes `json:"Attributes"`
}

func (r *BootOrderResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + bootOrderName
}

func BootOrderSchema() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"id": schema.StringAttribute{
			Computed:            true,
			MarkdownDescription: "ID of BIOS settings resource on iRMC.",
			Description:         "ID of BIOS settings resource on iRMC.",
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
			Required:            true,
			MarkdownDescription: "Control how system will be reset to finish boot order change (if host is powered on).",
			Description:         "Control how system will be reset to finish boot order change (if host is powered on).",
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
			Description:         "Timeout in seconds for boot order change to finish.",
			MarkdownDescription: "Timeout in seconds for boot order change to finish.",
			Validators: []validator.Int64{
				int64validator.AtLeast(240),
			},
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

func (r *BootOrderResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	tflog.Info(ctx, "resource-boot_order: create starts")

	// Read Terraform plan data into the model
	var plan models.BootOrderResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Provide synchronization
	var endpoint string = plan.RedfishServer[0].Endpoint.ValueString()
	var resource_name string = "resource-boot_order"
	mutexPool.Lock(ctx, endpoint, resource_name)
	defer mutexPool.Unlock(ctx, endpoint, resource_name)

	// Connect to service
	api, err := ConnectTargetSystem(r.p, &plan.RedfishServer)
	if err != nil {
		resp.Diagnostics.AddError("service error: ", err.Error())
		return
	}

	defer api.Logout()

	// Compare planned changes in boot order with current boot order options
	var plannedBootOrder []string
	diags = plan.BootOrder.ElementsAs(ctx, &plannedBootOrder, true)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Fetch current boot order and check if planned boot order
	// contains all requested devices
	currentBootOrder, diags := validateBootOrderPlan(api.Service, plannedBootOrder)
	resp.Diagnostics.Append(diags...)
	if diags.HasError() {
		return
	}

	// Apply boot order change
	diags = applyBootOrderPlan(api.Service, currentBootOrder, plannedBootOrder)
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

	tflog.Info(ctx, "resource-boot_order: create ends")
}

func (r *BootOrderResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	tflog.Info(ctx, "resource-boot_order: read starts")

	// Read Terraform prior state data into the model
	var currState, newState models.BootOrderResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &currState)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Connect to service
	api, err := ConnectTargetSystem(r.p, &currState.RedfishServer)
	if err != nil {
		resp.Diagnostics.AddError("service error: ", err.Error())
		return
	}

	defer api.Logout()
	diags := readCurrentBootOrder(api.Service, &newState)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	newState.JobTimeout = currState.JobTimeout
	newState.RedfishServer = currState.RedfishServer
	newState.SystemResetType = currState.SystemResetType
	newState.Id = types.StringValue(BIOS_SETTINGS_ENDPOINT)

	diags = resp.State.Set(ctx, &newState)
	resp.Diagnostics.Append(diags...)

	tflog.Info(ctx, "resource-boot_order: read ends")
}

func (r *BootOrderResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	tflog.Info(ctx, "resource-boot_order: update starts")

	// Read Terraform plan
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

	// Compare planned changes in boot order with current boot order options
	var plannedBootOrder []string
	diags = plan.BootOrder.ElementsAs(ctx, &plannedBootOrder, true)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Fetch current boot order and check if planned boot order
	// contains all requested devices
	currentBootOrder, diags := validateBootOrderPlan(api.Service, plannedBootOrder)
	resp.Diagnostics.Append(diags...)
	if diags.HasError() {
		return
	}

	// Apply boot order change
	diags = applyBootOrderPlan(api.Service, currentBootOrder, plannedBootOrder)
	resp.Diagnostics.Append(diags...)
	if diags.HasError() {
		return
	}

	diags = waitTillBootOrderApplied(ctx, api.Service, plan)
	resp.Diagnostics.Append(diags...)
	if diags.HasError() {
		return
	}

	plan.Id = types.StringValue("/redfish/v1/Systems/0/Bios/Settings")

	diags = resp.State.Set(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Info(ctx, "resource-boot_order: update ends")
}

func (r *BootOrderResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	tflog.Info(ctx, "resource-boot_order: delete starts")
	resp.State.RemoveResource(ctx)
	tflog.Info(ctx, "resource-boot_order: delete ends")
}

func (r *BootOrderResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	tflog.Info(ctx, "resource-boot_order: import starts")

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

	tflog.Info(ctx, "resource-boot_order: import ends")
}

func getDeviceNameFromStructureBootString(currentBootOrder []BootOrderEntry, structuredBootString string) string {
	for _, v := range currentBootOrder {
		if v.StructuredBootString == structuredBootString {
			return v.DeviceName
		}
	}

	return "NotFound" // FIXME/TODO
}

// applyBootOrderPlan tries to apply plannedBootOrder collecting required DeviceName
// from structured boot string which is part of plannedBootOrder into system
// pointed by service.
func applyBootOrderPlan(service *gofish.Service, currentBootOrder []BootOrderEntry, plannedBootOrder BootOrder) (diags diag.Diagnostics) {
	client := service.GetClient()
	res, err := client.Get(BIOS_SETTINGS_ENDPOINT)
	if err != nil {
		diags.AddError("Reading /redfish/v1/Systems/0/Bios/Settings failed", err.Error())
		return diags
	}

	res.Body.Close()

	var v [][]string
	for _, item := range plannedBootOrder {
		entry := make([]string, 0, 2)
		entry = append(entry, item)

		// DeviceName must be obtained from current boot order setup
		deviceName := getDeviceNameFromStructureBootString(currentBootOrder, item)
		entry = append(entry, deviceName)

		v = append(v, entry)
	}

	payload := map[string]interface{}{
		"Attributes": map[string]interface{}{
			PERSISTENT_BOOT_ORDER_KEY: v,
		},
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

// getBiosSettingsFutureAttributesNumber reads property Attributes from /Bios/Settings
// and returns number of elements inside of the property or error information during processing
// over diags.
func getBiosSettingsFutureAttributesNumber(service *gofish.Service) (length int, diags diag.Diagnostics) {
	client := service.GetClient()
	res, err := client.Get(BIOS_SETTINGS_ENDPOINT)
	if err != nil {
		diags.AddError("Reading /redfish/v1/Systems/0/Bios/Settings failed", err.Error())
		return 0, diags
	}

	defer res.Body.Close()

	var config BiosSettings
	bodyBytes, err := io.ReadAll(res.Body)
	if err != nil {
		diags.AddError("Reading body of /redfish/v1/Systems/0/Bios/Settings failed", err.Error())
		return 0, diags
	}

	err = json.Unmarshal(bodyBytes, &config)
	if err != nil {
		diags.AddError("Failed to unmarshal /redfish/v1/Systems/0/Bios/Settings response body", err.Error())
		return 0, diags
	}

	return len(config.Attributes), diags
}

// waitTillBootOrderApplied supervises applying boot order from plan
// and return possible errors during processing using diags.
func waitTillBootOrderApplied(ctx context.Context, service *gofish.Service, plan models.BootOrderResourceModel) (diags diag.Diagnostics) {
	poweredOn, err := isPoweredOn(service)
	if err != nil {
		diags.AddError("Could not retrieve current power state", err.Error())
		return diags
	}

	timeout := plan.JobTimeout.ValueInt64()
	var logMsg string = fmt.Sprintf("Process will wait with %d seconds timeout to finish", timeout)
	tflog.Info(ctx, logMsg)

	startTime := time.Now().Unix()

	if !poweredOn {
		err = changePowerState(service, true, timeout)
	} else {
		resetType := (redfish.ResetType)(plan.SystemResetType.ValueString())
		err = resetHost(service, resetType, timeout)
	}

	// Due to BIOS setting change it might happen that host will be powered off after
	// BIOS POST phase, so to not break the process the error must be omitted
	if err.Error() != "BIOS exited POST but host powered off" {
		diags.AddError("Host could not be powered on to finish BIOS settings", err.Error())
		return diags
	}

	if time.Now().Unix()-startTime > timeout {
		diags.AddError("Job timeout exceeded after reset/power on while operation has not finished", "Terminate")
		return diags
	}

	for {
		numberOfKeysInMap, diags := getBiosSettingsFutureAttributesNumber(service)
		if diags.HasError() {
			return diags
		}

		/*
		   At the moment the only way to check if process is finished is to check
		   Attributes parameter of /Bios/Settings. In case of parameters not yet applied,
		   it will contain only these which are planned to be applied. After process complete,
		   it will contain all writable properties. It's not best mechanism, but the only one known as of now
		*/
		if numberOfKeysInMap > 5 {
			var logMsg string = fmt.Sprintf("Number of keys %d", numberOfKeysInMap)
			tflog.Info(ctx, logMsg)
			break
		}

		time.Sleep(2 * time.Second)
		if time.Now().Unix()-startTime > timeout {
			diags.AddError("Job timeout exceeded while operation has not finished", "Terminate")
			return diags
		}
	}

	return diags
}

type BootEntry struct {
	StructuredBootString string
	DeviceName           string
}

func (be *BootEntry) UnmarshalJSON(data []byte) error {
	var v []interface{}
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}

	if val, ok := v[0].(string); ok {
		be.StructuredBootString = val
	}

	if val, ok := v[1].(string); ok {
		be.DeviceName = val
	}

	return nil
}

func isBootEntryInBootOrder(value string, bootOrder []BootOrderEntry) bool {
	for _, v := range bootOrder {
		if value == v.StructuredBootString {
			return true
		}
	}
	return false
}

func findAvailableAndNotPlannedBootEntries(currentBootOrder []BootOrderEntry, plannedBootOrder BootOrder) []string {
	mb := make(map[string]struct{}, len(plannedBootOrder))
	for _, x := range plannedBootOrder {
		mb[x] = struct{}{}
	}
	var diff []string
	for _, x := range currentBootOrder {
		if _, found := mb[x.StructuredBootString]; !found {
			diff = append(diff, x.StructuredBootString)
		}
	}
	return diff
}

// validateBootOrderPlan serves for validation of plannedBootOrder vs currently configuration boot order
// As a result it returns obtained currentBootOrder and diagnostic logs.
func validateBootOrderPlan(service *gofish.Service, plannedBootOrder BootOrder) (currentBootOrder []BootOrderEntry, diags diag.Diagnostics) {
	system, err := GetSystemResource(service)
	if err != nil {
		diags.AddError("Error while reading /Systems/0", err.Error())
		return currentBootOrder, diags
	}

	rBios, err := system.Bios()
	if err != nil {
		diags.AddError("Error while reading /Systems/0/Bios", err.Error())
		return currentBootOrder, diags
	}

	if len(rBios.Attributes) == 0 {
		diags.AddError("No BIOS data for BIOS attributes yet", rBios.ODataID)
		return currentBootOrder, diags
	}

	// Read current boot order
	if currentBootConfigOrder, persistentBiosConfigExists := rBios.Attributes[PERSISTENT_BOOT_ORDER_KEY]; persistentBiosConfigExists {
		bootOrderStr, _ := json.Marshal(currentBootConfigOrder)
		var bootOrderList []BootEntry
		if err := json.Unmarshal(bootOrderStr, &bootOrderList); err != nil {
			diags.AddError("PersistentBootConfigOrder could not be unmarshalled", err.Error())
			return currentBootOrder, diags
		}

		for _, item := range bootOrderList {
			var entry BootOrderEntry
			entry.DeviceName = item.DeviceName
			entry.StructuredBootString = item.StructuredBootString
			currentBootOrder = append(currentBootOrder, entry)
		}

		// If any planned option does not exist on currently configured boot order, raise error
		for _, v := range plannedBootOrder {
			if !isBootEntryInBootOrder(v, currentBootOrder) {
				var msg string = fmt.Sprintf("Entry '%s' is not on the list of supported boot entries for the system '%s'", v, currentBootOrder)
				diags.AddError("Planned changes for boot order did not pass validation", msg)
			}
		}

		if diags.HasError() {
			return currentBootOrder, diags
		}

		// If planned configuration does not contain all options for the system, stop
		if len(plannedBootOrder) != len(currentBootOrder) {
			var details string = fmt.Sprintf("Planned boot order has length of %d, while current length of %d",
				len(plannedBootOrder), len(currentBootOrder))
			diags.AddError("Planned boot order has different length than currently configured boot order", details)
			return currentBootOrder, diags
		}

		if diff := findAvailableAndNotPlannedBootEntries(currentBootOrder, plannedBootOrder); len(diff) > 0 {
			var details string = fmt.Sprintf("Planned boot order does not contain available boot options '%s'",
				strings.Join(diff, ""))
			diags.AddError("Planned boot order does not contain all available boot options", details)
			return currentBootOrder, diags
		}

		return currentBootOrder, diags
	} else {
		diags.AddError("Missing PersistentBootConfigOrder parameter in attribute", "Server returned unexpected content")
		return currentBootOrder, diags
	}
}

// readCurrentBootOrder reads currently configured boot order and save it to state.
func readCurrentBootOrder(service *gofish.Service, state *models.BootOrderResourceModel) (diags diag.Diagnostics) {
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

	if len(rBios.Attributes) == 0 {
		diags.AddError("No BIOS data for BIOS attributes yet", rBios.ODataID)
		return diags
	}

	// Read current boot order
	if currentBootConfigOrder, persistentBiosConfigExists := rBios.Attributes[PERSISTENT_BOOT_ORDER_KEY]; persistentBiosConfigExists {
		bootOrderStr, _ := json.Marshal(currentBootConfigOrder)
		var bootOrderList []BootEntry
		if err := json.Unmarshal(bootOrderStr, &bootOrderList); err != nil {
			diags.AddError("PersistentBootConfigOrder could not be unmarshalled", err.Error())
			return diags
		}

		bootOrder := []attr.Value{}
		for _, item := range bootOrderList {
			bootOrder = append(bootOrder, types.StringValue(item.StructuredBootString))
		}

		state.BootOrder, diags = types.ListValue(types.StringType, bootOrder)
		if diags.HasError() {
			return diags
		}
	}

	return diags
}
