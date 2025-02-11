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
	"net/http"
	"os"
	"terraform-provider-irmc-redfish/internal/models"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/stmcginnis/gofish"
)

const (
	UPLOAD_CERT_ENDPOINT = "/redfish/v1/Managers/iRMC/Oem/ts_fujitsu/iRMCConfiguration/Certificates/Actions/FTSCertificates.UploadSSLCertOrKey"
	VERIFY_CERT_ENDPOINT = "/redfish/v1/Managers/iRMC/Oem/ts_fujitsu/iRMCConfiguration/Certificates/Actions/FTSCertificates.VerifySSLCertKeyCompliance"
	CERT_ENDPOINT        = "/redfish/v1/Managers/iRMC/Oem/ts_fujitsu/iRMCConfiguration/Certificates"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &IrmcCertificateWebServerResource{}

func NewIrmcCertificateWebServerResource() resource.Resource {
	return &IrmcCertificateWebServerResource{}
}

// IrmcCertificateWebServerResource defines the resource implementation.
type IrmcCertificateWebServerResource struct {
	p *IrmcProvider
}

func (r *IrmcCertificateWebServerResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + certificateWebServer
}

func IrmcCertificateWebServerSchema() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"id": schema.StringAttribute{
			Computed:            true,
			MarkdownDescription: "ID of irmc web server certificate resource on iRMC.",
			Description:         "ID of irmc web server certificate resource on iRMC.",
		},
		"cert_private_key": schema.StringAttribute{
			Required:            true,
			MarkdownDescription: "Path to the private key (.pem file).",
			Description:         "Path to the private key (.pem file).",
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.RequiresReplaceIfConfigured(),
			},
		},
		"cert_public_key": schema.StringAttribute{
			Required:            true,
			MarkdownDescription: "Path to the public key (.pem file).",
			Description:         "Path to the public key (.pem file).",
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.RequiresReplaceIfConfigured(),
			},
		},
	}
}

func (r *IrmcCertificateWebServerResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "This resource is used to upload web server certificate in the IRMC.",
		Description:         "This resource is used to upload web server certificate in the IRMC.",
		Attributes:          IrmcCertificateWebServerSchema(),
		Blocks:              RedfishServerResourceBlockMap(),
	}
}

func (r *IrmcCertificateWebServerResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *IrmcCertificateWebServerResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	tflog.Info(ctx, "resource-certificate-web-server: Create starts")

	var plan models.CertificateWebServerResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Provide synchronization
	var endpoint string = plan.RedfishServer[0].Endpoint.ValueString()
	var resource_name string = "certificate_web_server"
	mutexPool.Lock(ctx, endpoint, resource_name)
	defer mutexPool.Unlock(ctx, endpoint, resource_name)

	api, err := ConnectTargetSystem(r.p, &plan.RedfishServer)
	if err != nil {
		resp.Diagnostics.AddError("Service Connection Error", err.Error())
		return
	}
	defer api.Logout()

	err = sendCertificateUpdate(api, plan.CertPublicKey.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to upload public certificate", err.Error())
		return
	}

	err = sendCertificateUpdate(api, plan.CertPrivateKey.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to upload private key", err.Error())
		return
	}

	err = verifyCertificateCompliance(api)
	if err != nil {
		resp.Diagnostics.AddError("Certificate verification failed", err.Error())
		return
	}

	err = restartIrmc(ctx, api, plan.RedfishServer, r.p)
	if err != nil {
		resp.Diagnostics.AddError("Failed to restart iRMC", err.Error())
		return
	}

	plan.Id = types.StringValue(CERT_ENDPOINT)
	diags = resp.State.Set(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Info(ctx, "resource-certificate-web-server: create ends")
}

// Read handles reading the resource state.
func (r *IrmcCertificateWebServerResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	tflog.Info(ctx, "resource-certificate-web-server: read starts")
	var state models.CertificateWebServerResourceModel

	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Save into State
	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	tflog.Info(ctx, "resource-certificate-web-server: read ends")
}

// Update modifies the resource state but returns an error if triggered, as updates are not supported.
func (r *IrmcCertificateWebServerResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// This function should not be called since updates are not supported; the resource should be recreated instead.
	resp.Diagnostics.AddError(
		"Unsupported Update Operation for IRMC Web Server Certificate",
		"The IRMC Web Server Certificate resource does not support in-place updates. It is intended to be destroyed and recreated if changes are required.",
	)
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *IrmcCertificateWebServerResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	tflog.Info(ctx, "resource-certificate-web-server: delete starts")
	// Delete State Data
	resp.State.RemoveResource(ctx)
	tflog.Info(ctx, "resource-certificate-web-server: delete ends")
}

func sendCertificateUpdate(api *gofish.APIClient, filePath string) error {
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("unable to open file %s: %w", filePath, err)
	}
	defer file.Close()

	payload := map[string]io.Reader{
		"data": file,
	}

	resp, err := api.Service.GetClient().PostMultipart(UPLOAD_CERT_ENDPOINT, payload)
	if err != nil {
		return fmt.Errorf("error sending certificate update: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to upload certificate, status: %d, response: %s", resp.StatusCode, string(body))
	}

	return nil
}

func verifyCertificateCompliance(api *gofish.APIClient) error {
	resp, err := api.Service.GetClient().Post(VERIFY_CERT_ENDPOINT, nil)
	if err != nil {
		return fmt.Errorf("error sending POST request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("unexpected response status, status code: %d, response: %s", resp.StatusCode, string(body))
	}

	var result struct {
		Compliant bool `json:"Compliant"`
	}

	err = json.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		return fmt.Errorf("failed to parse verification response: %w", err)
	}

	if !result.Compliant {
		return fmt.Errorf("certificate verification failed: non-compliant certificate")
	}

	return nil
}
