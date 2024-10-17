# irmc-redfish_boot_order (Resource)

The resource is used to control (Create, Read, Delete, Update and Import) boot order settings on Fujitsu server equipped with iRMC controller.
To apply boot order you must know all supported boot options. Current boot order configuration of a specific server can be obtained using property 
Attributes::PersistentBootConfigOrder of the endpoint: /redfish/v1/Systems/0/Bios


## Schema

### Required

- `boot_order` (List of String) Boot devices order in BIOS.
- `system_reset_type` (String) Control how system will be reset to finish boot order change (if host is powered on). Applicable values are: 'ForceRestart', 'GracefulRestart', 'PowerCycle'.

### Optional

- `job_timeout` (Number) Timeout in seconds for boot order change to finish (default 600s).
- `server` (Block List) List of server BMCs and their respective user credentials (see [below for nested schema](#nestedblock--server))

### Read-Only

- `id` (String) ID of BIOS settings resource on iRMC.

<a id="nestedblock--server"></a>
### Nested Schema for `server`

Required:

- `endpoint` (String) Server BMC IP address or hostname

Optional:

- `password` (String, Sensitive) User password for login
- `ssl_insecure` (Boolean) This field indicates whether the SSL/TLS certificate must be verified or not
- `username` (String) User name for login

## Import

The resource supports importing boot order configuration from a server.

To import boot config order, the following syntax is expected to be used:
```shell
terraform import irmc-redfish_boot_order.bo "{\"username\":\"<username>\",\"password\":\"<password>\",\"endpoint\":\"<endpoint>\",\"ssl_insecure\":<true/false>}"
```

If import will be executed successfully, you should be able to list state of the imported resource.
The following state allowes you to have control over the resource using Terraform.
To modify resource e.g.: change boot order, you should fill in resource terraform file and check with terraform apply if any differences
between state and plan are visible beside these ones which are requested.
