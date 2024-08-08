// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	//	"image/internal/imageutil"
	"terraform-provider-irmc-redfish/internal/models"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
//	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	//	"google.golang.org/grpc/internal/idle"

	"github.com/stmcginnis/gofish"
	"github.com/stmcginnis/gofish/redfish"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &VirtualMediaResource{}
var _ resource.ResourceWithImportState = &VirtualMediaResource{}

func NewVirtualMediaResource() resource.Resource {
    return &VirtualMediaResource{}
}

// VirtualMediaResource defines the resource implementation.
type VirtualMediaResource struct {
    p *IrmcProvider
}
const VMEDIA_ENDPOINT = "/redfish/v1/Managers/iRMC/VirtualMedia/" 

func (r *VirtualMediaResource) updateVirtualMediaState(response *redfish.VirtualMedia, plan models.VirtualMediaResourceModel) models.VirtualMediaResourceModel {
    var new_id strings.Builder
    new_id.WriteString(VMEDIA_ENDPOINT)
    new_id.WriteString(response.ID)

    return models.VirtualMediaResourceModel{
        Id: types.StringValue(new_id.String()),
        Image: types.StringValue(response.Image),
        Inserted: types.BoolValue(response.Inserted),
        TransferProtocolType: types.StringValue(string(response.TransferProtocolType)),
        WriteProtected: types.BoolValue(response.WriteProtected),
        RedfishServer: plan.RedfishServer,
    }
}

func (r *VirtualMediaResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + vmediaName
}

func VirtualMediaSchema() map[string]schema.Attribute {
    return map[string]schema.Attribute{
        "id": schema.StringAttribute{
            Computed:            true,
            MarkdownDescription: "",
            Description:         "",
        },
        "image": schema.StringAttribute{
            Required:            true,
            MarkdownDescription: "",
            Description:         "",
        },
        "inserted": schema.BoolAttribute{
            Computed:            true,
            Description:         "Describes whether virtual media is attached or detached",
            MarkdownDescription: "Describes whether virtual media is attached or detached",
        },
        "transfer_protocol_type": schema.StringAttribute{
            MarkdownDescription: "",
            Description:         "",
            Optional: true,
            Validators: []validator.String{
                stringvalidator.OneOf([]string{"CIFS", "HTTP", "HTTPS", "NFS"}...),
            },
        },
        "write_protected": schema.BoolAttribute{
            Optional:            true,
            Computed:            true,
            Description:         "Indicates whether the remote device media prevents writing to that media.",
            MarkdownDescription: "Indicates whether the remote device media prevents writing to that media.",
            Default:             booldefault.StaticBool(true),
        },
    }
}

func (r *VirtualMediaResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
    resp.Schema = schema.Schema{
        MarkdownDescription: "Virtual media resource",
        Description:         "It is used to configure virtual media on iRMC",
        Attributes:          VirtualMediaSchema(),
        Blocks:              RedfishServerResourceBlockMap(),
    }
}

func (r *VirtualMediaResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

type virtualMediaEnvironment struct {
    collection []*redfish.VirtualMedia
    client *gofish.APIClient
}

func (r *VirtualMediaResource) GetVirtualMediaEnvironment(rserver *[]models.RedfishServer) (virtualMediaEnvironment, diag.Diagnostics) {
    var env virtualMediaEnvironment
    var d diag.Diagnostics
    var manager []*redfish.Manager

    api, err := ConnectTargetSystem(r.p, rserver)
    if err != nil {
        d.AddError("Error while connecting to SUT", err.Error())
        return env, d
    }

    env.client = api

    manager, err = api.Service.Managers()
    if err != nil {
        d.AddError("Error when accessing Managers resource", err.Error())
        return env, d
    }

    vmediaCollection, err := manager[0].VirtualMedia()
    if err != nil {
        d.AddError("Could not retrieve vmedia collection from redfish API", err.Error())
        return env, d
    }

    if len(vmediaCollection) != 0 {
        env.collection = vmediaCollection
    }

    return env, d
}

func GetVirtualMedia(vmediaID string, vms []*redfish.VirtualMedia) (*redfish.VirtualMedia, error) {
    for _, v := range vms {
        if v.ID == vmediaID {
            return v, nil
        }
    }

    return nil, fmt.Errorf("virtual media with ID %s does not exist", vmediaID)
}

// WaitForMediaSuccessfullyMounted checks requested endpoint of given service
// until the endpoint will returned Inserted as true or counter will reach limit
func WaitForMediaSuccessfullyMounted(service *gofish.Service, endpoint string) (*redfish.VirtualMedia, error) {
    cnt := 20 // number of tries every second
    virtualMedia, err := redfish.GetVirtualMedia(service.GetClient(), endpoint)
    for cnt > 0 {
        if err != nil {
            return nil, fmt.Errorf("%d Could not read media state %s due to %w", cnt,  endpoint, err)
        }

        if virtualMedia.Inserted == true {
            break
        }

        time.Sleep(1*time.Second)
        cnt--

        virtualMedia, err = redfish.GetVirtualMedia(service.GetClient(), endpoint)
    }
    return virtualMedia, nil
}

func InsertMedia(ctx context.Context, id string, collection []*redfish.VirtualMedia, config redfish.VirtualMediaConfig, service *gofish.Service) (*redfish.VirtualMedia, error) {
    virtualMedia, err := GetVirtualMedia(id, collection)
    if err != nil {
        return nil, fmt.Errorf("Virtual media with ID %s does not exist", id)
    }

    if virtualMedia.Inserted {
        tflog.Error(ctx, "Media insert has been requested on endpoint which has already mounted media")
        return nil, err
    }
    
    err = virtualMedia.InsertMediaConfig(config)
    if err != nil {
        return nil, fmt.Errorf("Could not mount vmedia %s: %w", id, err)
    }

    virtualMedia, err = WaitForMediaSuccessfullyMounted(service, virtualMedia.ODataID)
    if err != nil {
        return nil, fmt.Errorf("Reading status of selected virtual media finished with error: %w", err)
    }

    return virtualMedia, nil
}

type VmediaImageType int
const (
    IMAGE_TYPE_UNKNOWN VmediaImageType = iota
    IMAGE_TYPE_ISO
    IMAGE_TYPE_IMG
)

func (r *VirtualMediaResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
    tflog.Info(ctx, "virtual-media: create starts")

    // Read Terraform plan data into the model
    var plan models.VirtualMediaResourceModel
    diags := req.Plan.Get(ctx, &plan)
    resp.Diagnostics.Append(diags...)
    if resp.Diagnostics.HasError() {
        return
    }

    // Validate required image and define under which index it could be tried to be mounted
    image := plan.Image.ValueString()
    var imageType VmediaImageType = IMAGE_TYPE_UNKNOWN
    redfish_index := "0";
    if (strings.HasSuffix(image, ".iso")) {
        imageType = IMAGE_TYPE_ISO
        redfish_index = "0"
    } else {
        if (strings.HasSuffix(image, ".img")) {
            imageType = IMAGE_TYPE_IMG
            redfish_index = "1"
        }
    }

    if (imageType == IMAGE_TYPE_UNKNOWN) {
        resp.Diagnostics.AddError("Image type format is not supported", "Only .iso and .img formats are supported")
        return
    }

    // Get SUT virtual media environment
    var env virtualMediaEnvironment
    var d diag.Diagnostics
    env, d = r.GetVirtualMediaEnvironment(&plan.RedfishServer)
    resp.Diagnostics = append(resp.Diagnostics, d...)
    if resp.Diagnostics.HasError() {
        return
    }

    defer env.client.Logout()
  
    // Construct request to insert media
    virtualMediaConfig := redfish.VirtualMediaConfig {
        Image: image,
        Inserted: plan.Inserted.ValueBool(),
        TransferProtocolType: redfish.TransferProtocolType(plan.TransferProtocolType.ValueString()),
        WriteProtected: plan.WriteProtected.ValueBool(),
    }

    // Look for slot corresponding to requested image type
    service, vmediaCollection := env.client.Service, env.collection
    for index := range vmediaCollection {
        if (vmediaCollection[index].ID == redfish_index) {
            
            vmedia, err := InsertMedia(ctx, vmediaCollection[index].ID, vmediaCollection, virtualMediaConfig, service)
            if err != nil {
                resp.Diagnostics.AddError("Error while inserting vmedia ", err.Error())
                return
            }

            if vmedia != nil {
                result := r.updateVirtualMediaState(vmedia, plan)
                diags = resp.State.Set(ctx, &result)
                resp.Diagnostics.Append(diags...)
                tflog.Info(ctx, "virtual-media: create ends")
                return
            }
        }
    }

    resp.Diagnostics.AddError("Error: there are no virtual media to mount", "Please detach media and try again")
    resp.Diagnostics.Append(diags...)
}

func (r *VirtualMediaResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
    tflog.Info(ctx, "virtual-media: read starts")

    // Read Terraform prior state data into the model
    var state models.VirtualMediaResourceModel
    resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
    if resp.Diagnostics.HasError() {
        return
    }

    // Connect to service
    api, err := ConnectTargetSystem(r.p, &state.RedfishServer)
    if err != nil {
        resp.Diagnostics.AddError("service error: ", err.Error())
        return
    }

    defer api.Logout()

    // Get information about virtual media slot into which the plan has been applied
    virtualMedia, err := redfish.GetVirtualMedia(api.Service.GetClient(), state.Id.ValueString())
    if err != nil {
        resp.Diagnostics.AddError("Virtual media does not exist: ", err.Error())
        return
    }

    if len(virtualMedia.Image) == 0 {
        return
    }

    // Save updated data into Terraform state
    new_state := r.updateVirtualMediaState(virtualMedia, state)
    resp.Diagnostics.Append(resp.State.Set(ctx, &new_state)...)
    tflog.Info(ctx, "virtual-media: read ends")
}

func (r *VirtualMediaResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
    tflog.Info(ctx, "virtual-media: update starts")

    // Read Terraform plan
    var plan models.VirtualMediaResourceModel
    resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
    if resp.Diagnostics.HasError() {
        return
    }
   
    // Read terraform state
    var state models.VirtualMediaResourceModel
    diags := req.State.Get(ctx, &state)
    resp.Diagnostics.Append(diags...)
    if resp.Diagnostics.HasError() {
        return
    }

    // Validate required image and define under which index it could be tried to be mounted
    image := plan.Image.ValueString()
    var imageType VmediaImageType = IMAGE_TYPE_UNKNOWN
    redfish_index := 0;
    if (strings.HasSuffix(image, ".iso")) {
        imageType = IMAGE_TYPE_ISO
        redfish_index = 0
    } else {
        if (strings.HasSuffix(image, ".img")) {
            imageType = IMAGE_TYPE_IMG
            redfish_index = 1
        }
    }

    redfish_index = redfish_index+1

    if (imageType == IMAGE_TYPE_UNKNOWN) {
        resp.Diagnostics.AddError("Image type format is not supported", "Only .iso and .img formats are supported")
        return
    }

    // Get information about current virtual media setup
    api, err := ConnectTargetSystem(r.p, &state.RedfishServer)
    if err != nil {
        resp.Diagnostics.AddError("Connection to service failed: ", err.Error())
        return
    }
    
    defer api.Logout()

    vmedia, err := redfish.GetVirtualMedia(api.Service.GetClient(), state.Id.ValueString())
    if err != nil {
        resp.Diagnostics.AddError("Virtual media resource does not exist: ", err.Error())
        return
    }

    if vmedia.Inserted {
        err = vmedia.EjectMedia()
        if err != nil {
            resp.Diagnostics.AddError("Error while ejecting media: ", err.Error())
            return
        }

        time.Sleep(2 * time.Second)
    }

    // Construct request to insert media
    virtualMediaConfig := redfish.VirtualMediaConfig {
        Image: image,
        Inserted: plan.Inserted.ValueBool(),
        TransferProtocolType: redfish.TransferProtocolType(plan.TransferProtocolType.ValueString()),
        WriteProtected: plan.WriteProtected.ValueBool(),
    }

    err = vmedia.InsertMediaConfig(virtualMediaConfig)
    if err != nil {
        resp.Diagnostics.AddError("Could not mount virtual media ", err.Error())
        return
    }

    vmedia, err = redfish.GetVirtualMedia(api.Service.GetClient(), state.Id.ValueString())
    if err != nil {
        resp.Diagnostics.AddError("Virtual media does not exist ", err.Error())
        return
    }

    // Save updated data into Terraform state
    result := r.updateVirtualMediaState(vmedia, state)
    diags = resp.State.Set(ctx, &result)
    resp.Diagnostics.Append(diags...)
    tflog.Info(ctx, "virtual-media: update ends")
}

func (r *VirtualMediaResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
    tflog.Info(ctx, "virtual-media: delete starts")

    // Read Terraform prior state data into the model
    var state models.VirtualMediaResourceModel
    diags := req.State.Get(ctx, &state)
    resp.Diagnostics.Append(diags...)
    if resp.Diagnostics.HasError() {
        return
    }

    // Get information about current virtual media setup
    api, err := ConnectTargetSystem(r.p, &state.RedfishServer)
    if err != nil {
        resp.Diagnostics.AddError("Connection to service failed: ", err.Error())
        return
    }

    defer api.Logout()

    vmedia, err := redfish.GetVirtualMedia(api.Service.GetClient(), state.Id.ValueString())
    if err != nil {
        resp.Diagnostics.AddError("Virtual media resource does not exist: ", err.Error())
        return
    }

    err = vmedia.EjectMedia()
    if err != nil {
        resp.Diagnostics.AddError("Virtual media eject finished with error: ", err.Error())
        return
    }

    // Backup state information
    result := r.updateVirtualMediaState(vmedia, state)
    diags = resp.State.Set(ctx, &result)
    resp.Diagnostics.Append(diags...)
    tflog.Info(ctx, "virtual_media: delete ends")
}

type VMediaImportConfig struct {
    ServerConfig
    ID string `json:"id"`
}

func (r *VirtualMediaResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
    tflog.Info(ctx, "virtual_media: import starts")

    var config VMediaImportConfig
    err := json.Unmarshal([]byte(req.ID), &config)
    if err != nil {
        resp.Diagnostics.AddError("Error while 2 unmarshalling import config", err.Error())
        return
    }

    server := models.RedfishServer {
        User: types.StringValue(config.Username),
        Password: types.StringValue(config.Password),
        Endpoint: types.StringValue(config.Endpoint),
        SslInsecure: types.BoolValue(config.SslInsecure),
    }

    creds := []models.RedfishServer{server}
    
    // Get SUT virtual media environment
    var env virtualMediaEnvironment
    var d diag.Diagnostics
    env, d = r.GetVirtualMediaEnvironment(&creds)
    resp.Diagnostics = append(resp.Diagnostics, d...)
    if resp.Diagnostics.HasError() {
        return
    }
    
    defer env.client.Logout()

    // In collection of vmedia from SUT, look for the one which is intended to be imported
    var vmedia *redfish.VirtualMedia
    for _, vm := range env.collection {
        if vm.ODataID == config.ID {
            vmedia = vm
        }
    }

    if vmedia == nil {
        resp.Diagnostics.AddError("Virtual media with ID "+config.ID+" does not exist.", "")
        return
    }

    result := r.updateVirtualMediaState(vmedia, models.VirtualMediaResourceModel{
        RedfishServer: creds,
    })
    diags := resp.State.Set(ctx, &result)
    resp.Diagnostics.Append(diags...)
    
    tflog.Info(ctx, "virtual_media: import ends")
}
