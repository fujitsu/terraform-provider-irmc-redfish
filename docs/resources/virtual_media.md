# irmc-redfish_virtual_media (Resource)

The resource is used to control (read, mount, unmount or modify) virtual media on Fujitsu server equipped with iRMC controller.

## Schema

### Required

- `image` (String) URI of the remote media to be used for mounting.
- `transfer_protocol_type` (String) Indicates protocol on which the transfer will be done.

### Optional

- `server` (Block List) List of server BMCs and their respective user credentials (see [below for nested schema](#nestedblock--server))

### Read-Only

- `id` (String) ID of virtual media resource on iRMC.
- `inserted` (Boolean) Describes whether virtual media is mounted or not.

<a id="nestedblock--server"></a>
### Nested Schema for `server`

Required:

- `endpoint` (String) Server BMC IP address or hostname
- `password` (String, Sensitive) User password for login
- `username` (String) User name for login

Optional:

- `ssl_insecure` (Boolean) This field indicates whether the SSL/TLS certificate must be verified or not
