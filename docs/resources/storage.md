---
page_title: "irmc-redfish_storage Resource - irmc-redfish"
subcategory: ""
description: |-
  The resource is used to control (read, modify or import) storage controller settings on Fujitsu server equipped with iRMC controller.
---

# irmc-redfish_storage (Resource)

The resource is used to control (read, modify or import) storage controller settings on Fujitsu server equipped with iRMC controller.
The resource operation is based on storage controller serial number which is unique per controller.
Please remember that not all properties defined in this resource and their possible values will be acceptable for every controller
and in every hardware configuration. Provider implementation applies only these properties to controller, which are requested in plan.


## Schema

### Required

- `storage_controller_serial_number` (String) Serial number of storage controller.

### Optional

- `auto_rebuild_enabled` (Boolean) Auto rebuild enabled.
- `bgi_rate` (Number) BGI rate percent (range 0-100).
- `bios_continue_on_error` (String) BIOS continue on error (available values: StopOnErrors, PauseOnErrors, IgnoreErrors, SafeModeOnErrors).
- `bios_status` (Boolean) BIOS status.
- `coercion_mode` (String) Coercion mode (available values: None, Coerce128MiB, Coerce1GiB).
- `job_timeout` (Number) Job timeout in seconds.
- `mdc_abort_on_error_enabled` (Boolean) MDC abort on error enabled.
- `mdc_rate` (Number) MDC rate percent (range 0-100).
- `mdc_schedule_mode` (String) MDC schedule mode (available values: Disabled, Sequential, Concurrent).
- `migration_rate` (Number) Migration rate percent (range 0-100).
- `patrol_read` (String) Patrol read (available values: Automatic, Enabled, Disabled, Manual).
- `patrol_read_rate` (Number) Patrol read rate percent (range 0-100).
- `patrol_read_recovery_support` (Boolean) Patrol read recovery support enabled.
- `rebuild_rate` (Number) Rebuild rate percent (range 0-100).
- `server` (Block List) List of server BMCs and their respective user credentials (see [below for nested schema](#nestedblock--server))
- `spindown_delay` (Number) Spindown delay (range 30-1440).
- `spindown_hotspare_enabled` (Boolean) Spindown hotspare enabled.
- `spindown_unconfigured_drive_enabled` (Boolean) Spindown unconfigured drive enabled.
- `spinup_delay` (Number) Spinup delay (range 0-6).

### Read-Only

- `id` (String) Endpoint of storage controller represented by serial number.

<a id="nestedblock--server"></a>
### Nested Schema for `server`

Required:

- `endpoint` (String) Server BMC IP address or hostname

Optional:

- `password` (String, Sensitive) User password for login
- `ssl_insecure` (Boolean) This field indicates whether the SSL/TLS certificate must be verified or not
- `username` (String) User name for login
