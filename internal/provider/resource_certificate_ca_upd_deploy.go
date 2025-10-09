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
	"terraform-provider-irmc-redfish/internal/validators"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/stmcginnis/gofish"
)

const (
	CERTIFICATE_UPLOAD_TYPE      = "certificate_upload_type"
	CERTIFICATE_UPLOAD_TYPE_FILE = "File"
	CERTIFICATE_UPLOAD_TYPE_TEXT = "Text"
)

type certCaUpdDeployEndpoints struct {
	certificateEndpoint string
}

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &IrmcCertificateCaUpdDeployResource{}

func NewIrmcCertificateCaUpdDeployResource() resource.Resource {
	return &IrmcCertificateCaUpdDeployResource{}
}

// IrmcCertificateCaUpdDeployResource defines the resource implementation.
type IrmcCertificateCaUpdDeployResource struct {
	p *IrmcProvider
}

func (r *IrmcCertificateCaUpdDeployResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + certificateCaUpdDeploy
}

func IrmcCertificateCaUpdDeploySchema() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"id": schema.StringAttribute{
			Optional:            true,
			Computed:            true,
			MarkdownDescription: "ID of irmc CA certificate for update deployment resource on iRMC.",
			Description:         "ID of irmc CA certificate for update deployment resource on iRMC.",
		},
		"certificate_upload_type": schema.StringAttribute{
			MarkdownDescription: "Method of uploading the certificate. Accepted values are `File` or `Text`.",
			Description:         "Method of uploading the certificate. Accepted values are `File` or `Text`.",
			Required:            true,
			Validators: []validator.String{
				stringvalidator.OneOf([]string{
					CERTIFICATE_UPLOAD_TYPE_FILE,
					CERTIFICATE_UPLOAD_TYPE_TEXT,
				}...),
			},
		},
		"certificate_file": schema.StringAttribute{
			MarkdownDescription: "Local file path for the certificate if `certificate_upload_type` is `File`.",
			Description:         "Local file path for the certificate if `certificate_upload_type` is `File`.",
			Optional:            true,
			Computed:            true,
			Default:             stringdefault.StaticString(""),
			Validators: []validator.String{
				validators.ChangeToRequired(CERTIFICATE_UPLOAD_TYPE, CERTIFICATE_UPLOAD_TYPE_FILE),
			},
		},
		"certificate_text": schema.StringAttribute{
			MarkdownDescription: "Certificate content in plain text, if `certificate_upload_type` is `Text`.",
			Description:         "Certificate content in plain text, if `certificate_upload_type` is `Text`.",
			Optional:            true,
			Computed:            true,
			Default:             stringdefault.StaticString(""),
			Validators: []validator.String{
				validators.ChangeToRequired(CERTIFICATE_UPLOAD_TYPE, CERTIFICATE_UPLOAD_TYPE_TEXT),
			},
		},
	}

}

func (r *IrmcCertificateCaUpdDeployResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "This resource is used to upload CA certificate for update an deployment in the IRMC.",
		Description:         "This resource is used to upload CA certificate for update an deployment in the IRMC.",
		Attributes:          IrmcCertificateCaUpdDeploySchema(),
		Blocks:              RedfishServerResourceBlockMap(),
	}
}

func (r *IrmcCertificateCaUpdDeployResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *IrmcCertificateCaUpdDeployResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	tflog.Info(ctx, "resource-certificate-ca-upd-deploy: create starts")
	// Get Plan Data
	var plan models.CertificateCaUpdDeployResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Provide synchronization
	var endpoint string = plan.RedfishServer[0].Endpoint.ValueString()
	var resource_name string = "certificate_ca_upd_deploy"
	mutexPool.Lock(ctx, endpoint, resource_name)
	defer mutexPool.Unlock(ctx, endpoint, resource_name)

	api, err := ConnectTargetSystem(r.p, &plan.RedfishServer)
	if err != nil {
		resp.Diagnostics.AddError("Service Connection Error", err.Error())
		return
	}
	defer api.Logout()

	isFsas, err := IsFsasCheck(ctx, api)
	if err != nil {
		resp.Diagnostics.AddError("Vendor Detection Failed", err.Error())
		return
	}

	endp := getCertCaUpdDeployEndpoints(isFsas)

	switch plan.CertificateUploadType.ValueString() {
	case CERTIFICATE_UPLOAD_TYPE_FILE:
		err := handleFileCertificate(api, &plan, endp.certificateEndpoint)
		if err != nil {
			resp.Diagnostics.AddError("File Certificate Upload failed.", err.Error())
			return
		}
	case CERTIFICATE_UPLOAD_TYPE_TEXT:
		err := handleTextCertificate(api, &plan, endp.certificateEndpoint)
		if err != nil {
			resp.Diagnostics.AddError("Text Certificate Upload failed.", err.Error())
			return
		}
	}

	diags = resp.State.Set(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Info(ctx, "resource-certificate-ca-upd-deploy: create ends")

}

// Read handles reading the resource state.
func (r *IrmcCertificateCaUpdDeployResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	tflog.Info(ctx, "resource-certificate-ca-upd-deploy: read starts")
	var state models.CertificateCaUpdDeployResourceModel

	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Save into State
	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	tflog.Info(ctx, "resource-certificate-ca-upd-deploy: read ends")
}

// Update modifies the resource state but returns an error if triggered, as updates are not supported.
func (r *IrmcCertificateCaUpdDeployResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// This function should not be called since updates are not supported; the resource should be recreated instead.
	resp.Diagnostics.AddError(
		"Unsupported Update Operation for IRMC CA Certificate for Update and Deployment",
		"The IRMC CA Certificate for Update and Deployment resource does not support in-place updates. It is intended to be destroyed and recreated if changes are required.",
	)
}

func (r *IrmcCertificateCaUpdDeployResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	tflog.Info(ctx, "resource-certificate-ca-upd-deploy: delete starts")

	var state models.CertificateCaUpdDeployResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	api, err := ConnectTargetSystem(r.p, &state.RedfishServer)
	if err != nil {
		resp.Diagnostics.AddError("Service Connection Error", err.Error())
		return
	}
	defer api.Logout()

	if state.Id.IsNull() || state.Id.ValueString() == "" {
		resp.Diagnostics.AddError("Missing Certificate ID", "Cannot delete certificate without a valid ID.")
		return
	}

	certURL := state.Id.ValueString()

	deleteRes, err := api.Service.GetClient().Delete(certURL)
	if err != nil {
		resp.Diagnostics.AddError("Failed to delete certificate", err.Error())
		return
	}
	defer deleteRes.Body.Close()

	if deleteRes.StatusCode != http.StatusOK && deleteRes.StatusCode != http.StatusNoContent {
		responseBody, _ := io.ReadAll(deleteRes.Body)
		resp.Diagnostics.AddError("Unexpected response status", fmt.Sprintf("Status code: %d, response: %s", deleteRes.StatusCode, string(responseBody)))
		return
	}

	resp.State.RemoveResource(ctx)

	tflog.Info(ctx, "resource-certificate-ca-upd-deploy: delete ends")
}

func handleFileCertificate(api *gofish.APIClient, plan *models.CertificateCaUpdDeployResourceModel, certificateEndpoint string) error {

	fileContent, err := os.ReadFile(plan.CertificateFile.ValueString())
	if err != nil {
		return fmt.Errorf("could not read certificate file: %w", err)
	}

	res, err := api.Post(certificateEndpoint, string(fileContent))
	if err != nil {
		return fmt.Errorf("failed to upload certificate file: %w", err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK && res.StatusCode != http.StatusAccepted && res.StatusCode != http.StatusCreated {
		responseBody, _ := io.ReadAll(res.Body)
		return fmt.Errorf("unexpected response status: %d, response body: %s", res.StatusCode, string(responseBody))
	}
	taskLocation := res.Header.Get(HTTP_HEADER_LOCATION)
	if taskLocation == "" {
		return fmt.Errorf("task Location Missing. Location header not found in response")
	}
	plan.Id = types.StringValue(taskLocation)
	return nil
}

func handleTextCertificate(api *gofish.APIClient, plan *models.CertificateCaUpdDeployResourceModel, certificateEndpoint string) error {

	certificateContent := plan.CertificateText.ValueString()
	if certificateContent == "" {
		return fmt.Errorf("certificate text is empty")
	}

	res, err := api.Post(certificateEndpoint, certificateContent)
	if err != nil {
		return fmt.Errorf("failed to upload certificate text: %w", err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK && res.StatusCode != http.StatusAccepted && res.StatusCode != http.StatusCreated {
		responseBody, _ := io.ReadAll(res.Body)
		return fmt.Errorf("unexpected response status: %d, response body: %s", res.StatusCode, string(responseBody))
	}
	taskLocation := res.Header.Get(HTTP_HEADER_LOCATION)
	if taskLocation == "" {
		return fmt.Errorf("task Location Missing. Location header not found in response")
	}
	plan.Id = types.StringValue(taskLocation)
	return nil
}

func getCertCaUpdDeployEndpoints(isFsas bool) certCaUpdDeployEndpoints {
	if isFsas {
		return certCaUpdDeployEndpoints{
			certificateEndpoint: fmt.Sprintf("/redfish/v1/Managers/iRMC/Oem/%s/iRMCConfiguration/CertificationAuthority", FSAS),
		}
	} else {
		return certCaUpdDeployEndpoints{
			certificateEndpoint: fmt.Sprintf("/redfish/v1/Managers/iRMC/Oem/%s/iRMCConfiguration/CertificationAuthority", TS_FUJITSU),
		}
	}
}
