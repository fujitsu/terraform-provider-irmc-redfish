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

	"terraform-provider-irmc-redfish/internal/models"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ datasource.DataSource = &IrmcAttributesDataSource{}

func NewIrmcAttributesDataSource() datasource.DataSource {
	return &IrmcAttributesDataSource{}
}

// IrmcAttributesDataSource defines the data source implementation.
type IrmcAttributesDataSource struct {
	p *IrmcProvider
}

func (d *IrmcAttributesDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + iRMCAttributes
}

func IrmcAttributesDataSourceSchema() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"attributes": schema.MapAttribute{
			Computed:            true,
			MarkdownDescription: "Map of iRMC settings attributes.",
			Description:         "Map of iRMC settings attributes.",
			ElementType:         types.StringType,
		},
	}
}

func (d *IrmcAttributesDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "This datasource is used to query iRMC attributes",
		Attributes:          IrmcAttributesDataSourceSchema(),
		Blocks:              RedfishServerDatasourceBlockMap(),
	}
}

func (d *IrmcAttributesDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *IrmcAttributesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	// Read Terraform configuration data into the model
	var data models.IrmcAttributesDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		tflog.Trace(ctx, "has error!")
		return
	}

	// Connect to service
	api, err := ConnectTargetSystem(d.p, &data.RedfishServer)
	if err != nil {
		resp.Diagnostics.AddError("service error: ", err.Error())
		return
	}

	diags := readIrmcAttributesSettingsToModel(ctx, api.Service, &data.Attributes, true)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
