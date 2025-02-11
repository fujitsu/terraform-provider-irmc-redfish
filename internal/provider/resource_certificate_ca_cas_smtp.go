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
	CERTIFICATE_CA_CAS_CMTP_ENDPOINT        = "/redfish/v1/Managers/iRMC/Oem/ts_fujitsu/iRMCConfiguration/Certificates"
	CERTIFICATE_CA_CAS_CMTP_UPLOAD_ENDPOINT = "/redfish/v1/Managers/iRMC/Oem/ts_fujitsu/iRMCConfiguration/Certificates/Actions/FTSCertificates.UploadCACertificate"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &IrmcCertificateCaCasSmtpResource{}

func NewIrmcCertificateCaCasSmtpResource() resource.Resource {
	return &IrmcCertificateCaCasSmtpResource{}
}

// IrmcCertificateCaCasSmtpResource defines the resource implementation.
type IrmcCertificateCaCasSmtpResource struct {
	p *IrmcProvider
}

func (r *IrmcCertificateCaCasSmtpResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + certificateCaCasSmtp
}

func IrmcCertificateCaCasSmtpSchema() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"id": schema.StringAttribute{
			Computed:            true,
			MarkdownDescription: "ID of irmc CA CAS and SMTP certificate resource on iRMC.",
			Description:         "ID of irmc CA CAS and SMTP certificate resource on iRMC.",
		},
		"certificate_ca_file": schema.StringAttribute{
			Required:            true,
			MarkdownDescription: "Path to the ca certificate file (.pem file).",
			Description:         "Path to the ca certificate file (.pem file).",
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.RequiresReplaceIfConfigured(),
			},
		},
	}
}

func (r *IrmcCertificateCaCasSmtpResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "This resource is used to upload CA CAS and SMTP certificate in the IRMC.",
		Description:         "This resource is used to upload CA CAS and SMTP certificate in the IRMC.",
		Attributes:          IrmcCertificateCaCasSmtpSchema(),
		Blocks:              RedfishServerResourceBlockMap(),
	}
}

func (r *IrmcCertificateCaCasSmtpResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *IrmcCertificateCaCasSmtpResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	tflog.Info(ctx, "resource-certificate-ca-cas-smtp: Create starts")

	var plan models.CertificateCaCasSmtpResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Provide synchronization
	var endpoint string = plan.RedfishServer[0].Endpoint.ValueString()
	var resource_name string = "certificate_ca_cas_smtp"
	mutexPool.Lock(ctx, endpoint, resource_name)
	defer mutexPool.Unlock(ctx, endpoint, resource_name)

	api, err := ConnectTargetSystem(r.p, &plan.RedfishServer)
	if err != nil {
		resp.Diagnostics.AddError("Service Connection Error", err.Error())
		return
	}
	defer api.Logout()

	err = caCertificateUpload(api, &plan)
	if err != nil {
		resp.Diagnostics.AddError("Failed to upload public certificate", err.Error())
		return
	}

	plan.Id = types.StringValue(CERT_ENDPOINT)
	diags = resp.State.Set(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Info(ctx, "resource-certificate-ca-cas-smtp: create ends")
}

// Read handles reading the resource state.
func (r *IrmcCertificateCaCasSmtpResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	tflog.Info(ctx, "resource-certificate-ca-cas-smtp: read starts")
	var state models.CertificateCaCasSmtpResourceModel

	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Save into State
	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	tflog.Info(ctx, "resource-certificate-ca-cas-smtp: read ends")
}

// Update modifies the resource state but returns an error if triggered, as updates are not supported.
func (r *IrmcCertificateCaCasSmtpResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// This function should not be called since updates are not supported; the resource should be recreated instead.
	resp.Diagnostics.AddError(
		"Unsupported Update Operation for IRMC CA CAS and SMTP certificate",
		"The IRMC CA CAS and SMTP certificate resource does not support in-place updates. It is intended to be destroyed and recreated if changes are required.",
	)
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *IrmcCertificateCaCasSmtpResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	tflog.Info(ctx, "resource-certificate-ca-cas-smtp: delete starts")
	// Delete State Data
	resp.State.RemoveResource(ctx)
	tflog.Info(ctx, "resource-certificate-ca-cas-smtp: delete ends")
}

func caCertificateUpload(api *gofish.APIClient, plan *models.CertificateCaCasSmtpResourceModel) error {
	file, err := os.Open(plan.CertificateCaFile.ValueString())
	if err != nil {
		return fmt.Errorf("unable to open file %s: %w", plan.CertificateCaFile.ValueString(), err)
	}
	defer file.Close()

	payload := map[string]io.Reader{
		"data": file,
	}

	resp, err := api.Service.GetClient().PostMultipart(CERTIFICATE_CA_CAS_CMTP_UPLOAD_ENDPOINT, payload)
	if err != nil {
		return fmt.Errorf("error sending certificate upload: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to upload certificate, status: %d, response: %s", resp.StatusCode, string(body))
	}

	plan.Id = types.StringValue(CERTIFICATE_CA_CAS_CMTP_ENDPOINT)
	return nil
}
