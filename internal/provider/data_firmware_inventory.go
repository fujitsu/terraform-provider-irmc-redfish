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

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/stmcginnis/gofish"
)

const (
	FIRMWARE_INVENTORY_ENDPOINT = "/redfish/v1/UpdateService/FirmwareInventory"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ datasource.DataSource = &IrmcFirmwareInventoryDataSource{}

func NewFirmwareInventoryDataSource() datasource.DataSource {
	return &IrmcFirmwareInventoryDataSource{}
}

// IrmcFirmwareInventoryDataSource defines the data source implementation.
type IrmcFirmwareInventoryDataSource struct {
	p *IrmcProvider
}

func (d *IrmcFirmwareInventoryDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + firmwareInventory
}

func IrmcFirmwareInventorySchema() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"id": schema.StringAttribute{
			Computed:    true,
			Description: "ID of the firmware inventory.",
		},
		"inventory": schema.ListNestedAttribute{
			Computed: true,
			NestedObject: schema.NestedAttributeObject{
				Attributes: map[string]schema.Attribute{
					"odata_id": schema.StringAttribute{
						Computed:    true,
						Description: "OData ID of the firmware member.",
					},
					"id": schema.StringAttribute{
						Computed:    true,
						Description: "ID of the firmware member.",
					},
					"name": schema.StringAttribute{
						Computed:    true,
						Description: "Name of the firmware.",
					},
					"software_id": schema.StringAttribute{
						Computed:    true,
						Description: "Software ID of the firmware.",
					},
					"updateable": schema.BoolAttribute{
						Computed:    true,
						Description: "Indicates if the firmware is updateable.",
					},
					"version": schema.StringAttribute{
						Computed:    true,
						Description: "Version of the firmware.",
					},
					"state": schema.StringAttribute{
						Computed:    true,
						Description: "State of the firmware.",
					},
					"health": schema.StringAttribute{
						Computed:    true,
						Description: "Health status of the firmware.",
					},
				},
			},
		},
	}
}

func (d *IrmcFirmwareInventoryDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Firmware inventory data source",
		Attributes:          IrmcFirmwareInventorySchema(),
		Blocks:              RedfishServerDatasourceBlockMap(),
	}
}

func (d *IrmcFirmwareInventoryDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	p, ok := req.ProviderData.(*IrmcProvider)

	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *http.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)

		return
	}

	d.p = p
}

func (d *IrmcFirmwareInventoryDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {

	tflog.Info(ctx, "data-firmware-inventory: read starts")

	// Read Terraform configuration data into the model
	var data models.FirmwareInventory
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		tflog.Error(ctx, "has error!")
		return
	}

	api, err := ConnectTargetSystem(d.p, &data.RedfishServer)
	if err != nil {
		resp.Diagnostics.AddError("Service Connection Error", err.Error())
		return
	}
	defer api.Logout()

	members, err := GetFirmwareInventoryList(api)
	if err != nil {
		resp.Diagnostics.AddError("Error Getting Firmware Inventories", err.Error())
		return
	}
	data.ID = types.StringValue(FIRMWARE_INVENTORY_ENDPOINT)
	data.Inventory = members

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)

	tflog.Info(ctx, "data-firmware-inventory: read ends")
}

func GetFirmwareInventoryList(api *gofish.APIClient) ([]models.Inventory, error) {

	client := api.Service.GetClient()

	res, err := client.Get(FIRMWARE_INVENTORY_ENDPOINT)
	if err != nil {
		return nil, fmt.Errorf("error getting firmware inventory list: %w", err)
	}

	defer CloseResource(res.Body)

	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", res.StatusCode)
	}

	var firmwareList struct {
		Members []struct {
			OdataID string `json:"@odata.id"`
		} `json:"Members"`
	}

	err = json.NewDecoder(res.Body).Decode(&firmwareList)
	if err != nil {
		return nil, fmt.Errorf("error parsing firmware inventory list: %w", err)
	}

	var members []models.Inventory

	for _, member := range firmwareList.Members {
		detail, err := GetFirmwareInventoryDetail(api, member.OdataID)
		if err != nil {
			return nil, fmt.Errorf("error getting firmware inventory detail: %w", err)
		}
		members = append(members, detail)
	}

	return members, nil
}

func GetFirmwareInventoryDetail(api *gofish.APIClient, endpoint string) (models.Inventory, error) {
	client := api.Service.GetClient()
	res, err := client.Get(endpoint)
	if err != nil {
		return models.Inventory{}, fmt.Errorf("error getting firmware inventory detail: %w", err)
	}

	defer CloseResource(res.Body)

	if res.StatusCode != http.StatusOK {
		return models.Inventory{}, fmt.Errorf("unexpected status code: %d", res.StatusCode)
	}

	var detail struct {
		OdataID    string `json:"@odata.id"`
		Id         string `json:"Id"`
		Name       string `json:"Name"`
		SoftwareId string `json:"SoftwareId"`
		Updateable bool   `json:"Updateable"`
		Version    string `json:"Version"`
		Status     struct {
			State  string `json:"State"`
			Health string `json:"Health"`
		} `json:"Status"`
	}

	err = json.NewDecoder(res.Body).Decode(&detail)
	if err != nil {
		return models.Inventory{}, fmt.Errorf("error parsing firmware inventory detail: %w", err)
	}

	return models.Inventory{
		OdataID:    types.StringValue(detail.OdataID),
		Id:         types.StringValue(detail.Id),
		Name:       types.StringValue(detail.Name),
		SoftwareId: types.StringValue(detail.SoftwareId),
		Updateable: types.BoolValue(detail.Updateable),
		Version:    types.StringValue(detail.Version),
		State:      types.StringValue(detail.Status.State),
		Health:     types.StringValue(detail.Status.Health),
	}, nil
}
