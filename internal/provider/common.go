package provider

import (
        "errors"
        "fmt"
        "log"
	"terraform-provider-irmc-redfish/internal/models"

	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	datasourceSchema "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	resourceSchema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/stmcginnis/gofish"
)

const (
	redfishServerMD string = "List of server BMCs and their respective user credentials"
    vmediaName string = "virtual_media"
)

// RedfishServerDatasourceSchema to construct schema of redfish server
func RedfishServerDatasourceSchema() map[string]datasourceSchema.Attribute {
	return map[string]datasourceSchema.Attribute{
        "username": datasourceSchema.StringAttribute{
			Optional:    true,
			Description: "User name for login",
		},
		"password": datasourceSchema.StringAttribute{
			Optional:    true,
			Description: "User password for login",
			Sensitive:   true,
		},
		"endpoint": datasourceSchema.StringAttribute{
			Required:    true,
			Description: "Server BMC IP address or hostname",
		},
		"ssl_insecure": datasourceSchema.BoolAttribute{
			Optional:    true,
			Description: "This field indicates whether the SSL/TLS certificate must be verified or not",
		},
	}
}

func RedfishServerSchema() map[string]resourceSchema.Attribute {
	return map[string]resourceSchema.Attribute{
        "username": resourceSchema.StringAttribute{
			Optional:    true,
			Description: "User name for login",
		},
		"password": resourceSchema.StringAttribute{
			Optional:    true,
			Description: "User password for login",
			Sensitive:   true,
		},
		"endpoint": resourceSchema.StringAttribute{
			Required:    true,
			Description: "Server BMC IP address or hostname",
		},
		"ssl_insecure": resourceSchema.BoolAttribute{
			Optional:    true,
			Description: "This field indicates whether the SSL/TLS certificate must be verified or not",
		},
	}
}

// RedfishServerDatasourceBlockMap to construct common lock map for data sources
func RedfishServerDatasourceBlockMap() map[string]datasourceSchema.Block {
	return map[string]datasourceSchema.Block{
        "server": datasourceSchema.ListNestedBlock{
			MarkdownDescription: redfishServerMD,
			Description:         redfishServerMD,
			Validators: []validator.List{
				listvalidator.SizeAtMost(1),
				listvalidator.IsRequired(),
			},
			NestedObject: datasourceSchema.NestedBlockObject{
				Attributes: RedfishServerDatasourceSchema(),
			},
		},
	}
}

func RedfishServerResourceBlockMap() map[string]resourceSchema.Block {
        return map[string]resourceSchema.Block{
                "server": resourceSchema.ListNestedBlock{
                        MarkdownDescription: redfishServerMD,
			            Description:         redfishServerMD,
                        Validators: []validator.List{
                                listvalidator.SizeAtMost(1),
                                listvalidator.IsRequired(),
                        },
                        NestedObject: resourceSchema.NestedBlockObject{
                                Attributes: RedfishServerSchema(),
                        },
                },
        }
}

func NewConfig(pconfig *IrmcProvider, rserver *[]models.RedfishServer) (*gofish.Service, error) {
	if len(*rserver) == 0 {
		return nil, fmt.Errorf("no provider block was found")
	}

	// first redfish server block
	if len(*rserver) == 0 {
		return nil, errors.New("redfish server config not present")
	}
	rserver1 := (*rserver)[0]
	var redfishClientUser, redfishClientPass string

	if len(rserver1.User.ValueString()) > 0 {
		redfishClientUser = rserver1.User.ValueString()
	} else if len(pconfig.Username) > 0 {
		redfishClientUser = pconfig.Username
	} else {
		return nil, fmt.Errorf("error. Either provide username at provider level or resource level. Please check your configuration")
	}

	if len(rserver1.Password.ValueString()) > 0 {
		redfishClientPass = rserver1.Password.ValueString()
	} else if len(pconfig.Password) > 0 {
		redfishClientPass = pconfig.Password
	} else {
		return nil, fmt.Errorf("error. Either provide password at provider level or resource level. Please check your configuration")
	}

	if len(redfishClientUser) == 0 || len(redfishClientPass) == 0 {
		return nil, fmt.Errorf("error. Either Redfish client username or password has not been set. Please check your configuration")
	}

	clientConfig := gofish.ClientConfig{
		Endpoint:  rserver1.Endpoint.ValueString(),
		Username:  redfishClientUser,
		Password:  redfishClientPass,
		BasicAuth: true,
		Insecure:  rserver1.SslInsecure.ValueBool(),
	}
	api, err := gofish.Connect(clientConfig)
	if err != nil {
		return nil, fmt.Errorf("error connecting to redfish API: %w", err)
	}
	log.Printf("Connection with the redfish endpoint %v was sucessful\n", rserver1.Endpoint.ValueString())
	return api.Service, nil
}
