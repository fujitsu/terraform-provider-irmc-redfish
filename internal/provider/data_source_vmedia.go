// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

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
var _ datasource.DataSource = &IrmcVirtualMediaDataSource{}

func NewVirtualMediaDataSource() datasource.DataSource {
	return &IrmcVirtualMediaDataSource{}
}

// IrmcVirtualMediaDataSource defines the data source implementation.
type IrmcVirtualMediaDataSource struct {
	p *IrmcProvider
}

func (d *IrmcVirtualMediaDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + vmediaName
}

func VirtualMediaDataSourceSchema() map[string]schema.Attribute {
    return map[string]schema.Attribute {
        "virtual_media": schema.ListNestedAttribute{
            MarkdownDescription: "List of virtual media slots available on the system",
            Computed:            true,
            NestedObject: schema.NestedAttributeObject{
                Attributes: map[string]schema.Attribute{
                    "odata_id": schema.StringAttribute{
                        Computed:    true,
                        Description: "ODataId of virtual media resource",
                    },
                    "id": schema.StringAttribute{
                        Computed:    true,
                        Description: "Id of the virtual media resource",
                    },
                },
            },
        },
    }
}

func (d *IrmcVirtualMediaDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Virtual media data source",
		Attributes: VirtualMediaDataSourceSchema(), 
        Blocks: RedfishServerDatasourceBlockMap(),
	}
}

func (d *IrmcVirtualMediaDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *IrmcVirtualMediaDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	// Read Terraform configuration data into the model
	var data models.VirtualMediaDataSource
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

    // And look for virtual media resources
    managers, err := api.Service.Managers()
    if err != nil {
        resp.Diagnostics.AddError("Could not connect to the service: ", err.Error())
        return
    }

    vmedia_collection, err := managers[0].VirtualMedia()
    if err != nil {
        resp.Diagnostics.AddError("Virtual media does not exist: ", err.Error())
        return
    }

    if len(vmedia_collection) == 0 {
//        resp.Diagnostics.AddWarning("Virtual media collection is empty", "")
//        return
    }

    // Browse collection of vmedia and store its values
    for _, vmedia := range vmedia_collection {
        var found_vmedia models.VirtualMediaData
        found_vmedia.Id = types.StringValue(vmedia.ID)
        found_vmedia.ODataId = types.StringValue(vmedia.ODataID)

        data.VirtualMediaData = append(data.VirtualMediaData, found_vmedia)
    }

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
