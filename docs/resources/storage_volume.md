# irmc-redfish_storage_volume (Resource)

This resource is used to manipulate (Create, Read, Delete, Update and Import) logical volumes of iRMC system.
Please remember that every RAID controller might have its own specific behavior and allowed values for specific properties
depending on BBU installation status, types of disks, RAID type etc.
To facilitate process of volume creation for particular controller and situation it is recommended to check the following entries.

RAID controller serial number can be obtained by reading property SerialNumber in resource:
- /redfish/v1/Systems/0/Storage/<controllerId>

Every RAID controller presents its capabilities (supported RAID types etc.) in the following resource:
- /redfish/v1/Systems/0/Storage/<controllerId>/Oem/ts_fujitsu/RAIDCapabilities
- /redfish/v1/Systems/0/Storage/<controllerId>/Oem/Fsas/RAIDCapabilities


## Schema

### Required

- `optimum_io_size_bytes` (Number) Optimum IO size bytes (65536. 131072, 262144, 524288, 1048576).
- `physical_drives` (List of String) List of slot locations of disks used for volume creation.
- `raid_type` (String) RAID volume type depending on controller itself (RAID0, RAID1, RAID1E, RAID10, RAID5, RAID50, RAID6, RAID60).
- `storage_controller_serial_number` (String) Serial number of storage controller.

### Optional

- `capacity_bytes` (Number) Volume capacity in bytes. If not specified during creation, volume will have maximum size calculated from chosen disks.
- `drive_cache_mode` (String) Drive cache mode of volume (Enabled, Disabled, Unchanged).
- `init_mode` (String) Initialize mode for new volume (None, Fast, Normal).
- `job_timeout` (Number) Job timeout in seconds.
- `name` (String) Volume name
- `read_mode` (Attributes) (see [below for nested schema](#nestedatt--read_mode))
- `server` (Block List) List of server BMCs and their respective user credentials (see [below for nested schema](#nestedblock--server))
- `write_mode` (Attributes) (see [below for nested schema](#nestedatt--write_mode))

### Read-Only

- `id` (String) Id of handled volume

<a id="nestedatt--read_mode"></a>
### Nested Schema for `read_mode`

Optional:

- `requested` (String) Requested read mode of a created volume (Adaptive, NoReadAhead, ReadAhead).

Read-Only:

- `actual` (String) Actual read mode of a created volume.


<a id="nestedblock--server"></a>
### Nested Schema for `server`

Required:

- `endpoint` (String) Server BMC IP address or hostname

Optional:

- `password` (String, Sensitive) User password for login
- `ssl_insecure` (Boolean) This field indicates whether the SSL/TLS certificate must be verified or not
- `username` (String) User name for login

<a id="nestedatt--write_mode"></a>
### Nested Schema for `write_mode`

Optional:

- `requested` (String) Requested Write mode of a created volume (WriteBack, AlwaysWriteBack, WriteThrough).

Read-Only:

- `actual` (String) Actual Write mode of a created volume.


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
