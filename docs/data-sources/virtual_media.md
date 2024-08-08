# irmc-redfish_virtual_media (Data Source)

Virtual media data source

## Schema

### Optional

- `server` (Block List) List of server BMCs and their respective user credentials (see [below for nested schema](#nestedblock--server))

### Read-Only

- `virtual_media` (Attributes List) List of virtual media slots available on the system (see [below for nested schema](#nestedatt--virtual_media))

<a id="nestedblock--server"></a>
### Nested Schema for `server`

Required:

- `endpoint` (String) Server BMC IP address or hostname

Optional:

- `password` (String, Sensitive) User password for login
- `ssl_insecure` (Boolean) This field indicates whether the SSL/TLS certificate must be verified or not
- `username` (String) User name for login


<a id="nestedatt--virtual_media"></a>
### Nested Schema for `virtual_media`

Read-Only:

- `id` (String) Id of the virtual media resource
- `odata_id` (String) ODataId of virtual media resource
