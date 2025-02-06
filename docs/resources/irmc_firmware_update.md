<!--
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
-->

# irmc-redfish_irmc_firmware_update (Resource)

This resource is used to update the IRMC firmware.


## Schema

### Required

- `update_type` (String) Specifies the type of IRMC firmware update. Available options are: `File`, `TFTP`, and `MemoryCard`.

### Optional

- `id` (String) ID of the IRMC firmware update resource. Generated automatically by the system.
- `irmc_boot_selector` (String) Boot selector for the update. Possible options are: `Auto`, `LowFWImage`, `HighFWImage`, `OldestFW`, `MostRecentProgrammedFW`, and `LeastRecentProgrammedFW`. Default value: `Auto`:
                        "Auto":"Automatic - firmware with highest firmware version",
                        "LowFWImage":"Low firmware image",
                        "HighFWImage":"High firmware image",
                        "OldestFW":"Firmware with oldest firmware version",
                        "MostRecentProgrammedFW":"Most recently programmed firmware",
                        "LeastRecentProgrammedFW":"Least recently programmed firmware"
- `irmc_flash_selector` (String) Flash selector for the update. Possible options are: `Auto`, `LowFWImage`, and `HighFWImage`. Default value: `Auto`:
                        "Auto":"Automatic - inactive firmware image",
                        "LowFWImage":"Low firmware image",
                        "HighFWImage":"High firmware image"
- `irmc_path_to_binary` (String) Path to the binary firmware file to upload when `update_type` is `File`. Accepted format: absolute file path.
- `reset_irmc_after_update` (Boolean) Automatically reboot iRMC after flashing if set to `true`. If `false`, the user must reboot iRMC manually to complete the firmware update process. Default value: `true`.
- `server` (Block List) List of server BMCs and their respective user credentials (see [below for nested schema](#nestedblock--server))
- `tftp_server_addr` (String) Address of the TFTP server when `update_type` is `TFTP`. Accepted format: valid IP address or hostname.
- `tftp_update_file` (String) Path to the firmware file on the TFTP server when `update_type` is `TFTP`. Accepted format: relative file path (e.g., `/path/to/firmware.bin`).
- `update_timeout` (Number) Maximum duration (in seconds) to wait for the Firmware Update operation to finish before aborting. This does not include the time required for iRMC availability after the update. Default value: `3000` seconds.

<a id="nestedblock--server"></a>
### Nested Schema for `server`

Required:

- `endpoint` (String) Server BMC IP address or hostname

Optional:

- `password` (String, Sensitive) User password for login
- `ssl_insecure` (Boolean) This field indicates whether the SSL/TLS certificate must be verified or not
- `username` (String) User name for login
