package provider

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"terraform-provider-irmc-redfish/internal/models"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	datasourceSchema "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	resourceSchema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/stmcginnis/gofish"
	"github.com/stmcginnis/gofish/redfish"
)

const (
	redfishServerMD        string = "List of server BMCs and their respective user credentials"
	vmediaName             string = "virtual_media"
	storageVolumeName      string = "storage_volume"
	irmcRestart            string = "irmc_reset"
	bootSourceOverrideName string = "boot_source_override"
	bootOrderName          string = "boot_order"
	biosName               string = "bios"
	userAccount            string = "user_account"
)

const (
	HTTP_HEADER_IF_MATCH = "If-Match"
	HTTP_HEADER_ETAG     = "ETag"
)

type ServerConfig struct {
	Username    string `json:"username"`
	Password    string `json:"password"`
	Endpoint    string `json:"endpoint"`
	SslInsecure bool   `json:"ssl_insecure"`
}

type CommonImportConfig struct {
	ServerConfig
	ID string `json:"id"`
}

// RedfishServerDatasourceSchema to construct schema of redfish server.
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

// RedfishServerDatasourceBlockMap to construct common lock map for data sources.
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

func ConnectTargetSystem(pconfig *IrmcProvider, rserver *[]models.RedfishServer) (*gofish.APIClient, error) {
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

	return api, nil
}

// GetSystemResource returns ComputerSystem resource from target defined by service.
func GetSystemResource(service *gofish.Service) (*redfish.ComputerSystem, error) {
	systems, err := service.Systems()
	if err != nil {
		return nil, err
	}

	for _, system := range systems {
		if system.ID == "0" { // at the moment only one specific 0 is supported
			return system, nil
		}
	}

	return nil, fmt.Errorf("Requested System resource has not been found on list")
}

func difference(a, b []string) []string {
	mb := make(map[string]struct{}, len(b))
	for _, x := range b {
		mb[x] = struct{}{}
	}
	var diff []string
	for _, x := range a {
		if _, found := mb[x]; !found {
			diff = append(diff, x)
		}
	}
	return diff
}

// IsTaskFinished returns information whether task state
// has been mapped to task finished state and the information
// is returned as boolean.
func IsTaskFinished(state redfish.TaskState) bool {
	switch state {
	case redfish.CompletedTaskState, redfish.ExceptionTaskState, redfish.CancelledTaskState, redfish.KilledTaskState:
		fallthrough
	case redfish.InterruptedTaskState, redfish.SuspendedTaskState:
		return true
	default:
		break
	}
	return false
}

// IsTaskFinishedSuccessfully returns information whether task state
// has been mapped to task finished successfully or not and the information
// is returned as boolean.
func IsTaskFinishedSuccessfully(state redfish.TaskState) bool {
	switch state {
	case redfish.CompletedTaskState:
		return true
	default:
		return false
	}
}

// FetchRedfishTaskLog tries to fetch logs of task pointed by location
// from system accessed by service. If logs content could not be accessed
// diags is filled with reason.
func FetchRedfishTaskLog(service *gofish.Service, location string) (logs []byte, diags diag.Diagnostics) {
	task_log_endpoint := location + "/Oem/ts_fujitsu/Logs"
	res, err := service.GetClient().Get(task_log_endpoint)
	if err != nil {
		diags.AddError("Error while reading task log endpoint", err.Error())
		return nil, diags
	}

	defer res.Body.Close()

	if res.StatusCode == http.StatusOK {
		bodyBytes, err := io.ReadAll(res.Body)
		if err != nil {
			diags.AddError("Error while reading task logs", err.Error())
			return nil, diags
		}

		return bodyBytes, diags
	} else {
		diags.AddError("Error while reading task logs", "Endpoint returned non 200 code")
		return nil, diags
	}
}

func retryConnectWithTimeout(ctx context.Context, pconfig *IrmcProvider, rserver *[]models.RedfishServer) (*gofish.APIClient, error) {
	startTime := time.Now()
	var apiClient *gofish.APIClient
	var err error
	timeout := 10 * time.Minute

	for time.Since(startTime) < timeout {
		apiClient, err = ConnectTargetSystem(pconfig, rserver)
		if err == nil {
			tflog.Info(ctx, "Successfully connected to the IRMC system.")
			return apiClient, nil
		}

		tflog.Warn(ctx, fmt.Sprintf("Failed to connect to the IRMC system: %s. Retrying in 30 seconds...", err.Error()))
		time.Sleep(30 * time.Second)
	}

	return nil, fmt.Errorf("connection timed out after 10 minutes: %w", err)
}
