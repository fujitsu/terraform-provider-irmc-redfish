package models

import (
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type RedfishServer struct {
	User        types.String `tfsdk:"username"`
	Password    types.String `tfsdk:"password"`
	Endpoint    types.String `tfsdk:"endpoint"`
	SslInsecure types.Bool   `tfsdk:"ssl_insecure"`
}
