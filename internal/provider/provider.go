// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	//	"net/http"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	//	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Ensure IrmcProvider satisfies various provider interfaces.
var _ provider.Provider = &IrmcProvider{}

//var _ provider.ProviderWithFunctions = &IrmcProvider{}

// IrmcProvider defines the provider implementation.
type IrmcProvider struct {
	// version is set to the provider version on release, "dev" when the
	// provider is built and ran locally, and "test" when running acceptance
	// testing.
	version  string
	Username string
	Password string
	//	Endpoint string
}

// IrmcProviderModel describes the provider data model.
type IrmcProviderModel struct {
	Username types.String `tfsdk:"username"`
	Password types.String `tfsdk:"password"`
	//	Endpoint types.String `tfsdk:"endpoint"`
}

func (p *IrmcProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "irmc-redfish_"
	resp.Version = p.version
}

func (p *IrmcProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"username": schema.StringAttribute{
				MarkdownDescription: "",
				Description:         "Username accessing Redfish API",
				Optional:            true,
			},
			"password": schema.StringAttribute{
				MarkdownDescription: "",
				Description:         "Password related to given user name accessing Redfish API",
				Optional:            true,
			},
			//			"endpoint": schema.StringAttribute{
			//				MarkdownDescription: "",
			//				Description:         "Redfish API address",
			//				Optional:            true,
			//			},
		},
	}
}

func (p *IrmcProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var data IrmcProviderModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Configuration values are now available.
	if data.Username.IsUnknown() {
		resp.Diagnostics.AddWarning(
			"Unable to create client as username is missing",
			"Cannot use unknown value",
		)
	}

	if data.Password.IsUnknown() {
		resp.Diagnostics.AddWarning(
			"Unable to create client as password is missing",
			"Cannot use unknown value",
		)
	}

	p.Username = data.Username.ValueString()
	p.Password = data.Password.ValueString()

	resp.ResourceData = p
	resp.DataSourceData = p

	tflog.Trace(ctx, "Finished configuring the provider")

	// Example client configuration for data sources and resources
	/*
		client := http.DefaultClient
		resp.DataSourceData = client
		resp.ResourceData = client
	*/
}

func (p *IrmcProvider) Resources(ctx context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewVirtualMediaResource,
		NewBootOrderResource,
	}
}

func (p *IrmcProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewVirtualMediaDataSource,
	}
}

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &IrmcProvider{
			version: version,
		}
	}
}
