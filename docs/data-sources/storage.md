---
page_title: "irmc-redfish_storage Data Source - irmc-redfish"
subcategory: ""
description: |-
  Storage data source
---

# irmc-redfish_storage (Data Source)

Storage data source


## Schema

### Required

- `storage_controller_serial_number` (String) Serial number of storage controller.

### Optional

- `server` (Block List) List of server BMCs and their respective user credentials (see [below for nested schema](#nestedblock--server))

### Read-Only

- `auto_rebuild_enabled` (Boolean) Auto rebuild enabled.
- `bgi_rate` (Number) BGI rate percent.
- `bios_continue_on_error` (String) BIOS continue on error.
- `bios_status` (Boolean) BIOS status.
- `coercion_mode` (String) Coercion mode.
- `copyback_on_smart_error_support_enabled` (Boolean) Copyback on smart error support enabled.
- `copyback_on_ssd_smart_error_support_enabled` (Boolean) Copyback on SSD smart error support enabled.
- `copyback_support_enabled` (Boolean) Copyback support enabled.
- `id` (String) ID of BIOS settings resource on iRMC.
- `mdc_abort_on_error_enabled` (Boolean) MDC abort on error enabled.
- `mdc_rate` (Number) MDC rate percent.
- `mdc_schedule_mode` (String) MDC schedule mode.
- `migration_rate` (Number) Migration rate percent.
- `patrol_read` (String) Patrol read.
- `patrol_read_rate` (Number) Patrol read rate percent.
- `patrol_read_recovery_support` (Boolean) Patrol read recovery support enabled.
- `rebuild_rate` (Number) Rebuild rate percent.
- `spindown_delay` (Number) Spindown delay.
- `spindown_hotspare_enabled` (Boolean) Spindown hotspare enabled.
- `spindown_unconfigured_drive_enabled` (Boolean) Spindown unconfigured drive enabled.
- `spinup_delay` (Number) Spinup delay.

<a id="nestedblock--server"></a>
### Nested Schema for `server`

Required:

- `endpoint` (String) Server BMC IP address or hostname

Optional:

- `password` (String, Sensitive) User password for login
- `ssl_insecure` (Boolean) This field indicates whether the SSL/TLS certificate must be verified or not
- `username` (String) User name for login
