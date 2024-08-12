# irmc-redfish_storage_volume (Resource)

This resource is used to manipulate (Create, Read, Delete, Update and Import) logical volumes of iRMC system


## Schema

### Required

- `physical_drives` (List of String) Slot location of the disk
- `raid_type` (String) RAID volume type depending on controller itself
- `storage_controller_id` (String) Id of storage controller.

### Optional

- `capacity_bytes` (Number) Volume capacity in bytes. If not specified during creation, volume will have maximum size calculated from chosen disks.
- `drive_cache_mode` (String) Drive cache mode of volume.
- `init_mode` (String) Initialize mode for new volume.
- `name` (String) Volume name
- `optimum_io_size_bytes` (Number) Optimum IO size bytes
- `read_mode` (String) Read mode of volume.
- `server` (Block List) List of server BMCs and their respective user credentials (see [below for nested schema](#nestedblock--server))
- `write_mode` (String) Write mode of volume.

### Read-Only

- `id` (String) Id of handled volume

<a id="nestedblock--server"></a>
### Nested Schema for `server`

Required:

- `endpoint` (String) Server BMC IP address or hostname
- `password` (String, Sensitive) User password for login
- `username` (String) User name for login

Optional:

- `ssl_insecure` (Boolean) This field indicates whether the SSL/TLS certificate must be verified or not

## Import

The resource supports importing storage volume configuration from a server.
Existing volume collection from a specific server can be obtained using the following endpoints:
- /redfish/v1/Systems/0/Storage
- /redfish/v1/Systems/0/Storage/<storage_id>/Volumes

To import requested volume, the following syntax is expected to be used:
```shell
terraform import irmc-redfish_storage_volume.volume "{\"id\":\"<odata id of the volume>\",\"username\":\"<username>\",\"password\":\"<password>\",\"endpoint\":\"<endpoint>\",\"ssl_insecure\":<true/false>}"
```

If import will be executed successfully, you should be able to list state of the imported resource.
The following state allowes you to have control over the resource using Terraform.
To modify resource e.g.: change volume name, you should fill in resource terraform file and check with terraform apply if any differences
between state and plan are visible beside these ones which are requested.
