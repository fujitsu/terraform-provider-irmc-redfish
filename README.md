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

# Terraform Provider for iRMC Fujitsu Redfish API

The Terraform provider allows control and management over Fujitsu servers equipped with iRMC.

## Table of contents
* [License](#license)
* [Prerequisites](#prerequisites)
* [List of supported data sources](#list-of-supported-data-sources)
* [List of supported resources](#list-of-supported-resources)

## License
The provider is released and licensed under the MPL-2.0 license. See [License](LICENSE) for the full terms.

## Prerequisites
- [Terraform](https://developer.hashicorp.com/terraform/downloads) >= 1.0
- [Go](https://golang.org/doc/install) >= 1.21

## List of supported data sources
* [Virtual media](docs/data-sources/virtual_media.md)

## List of supported resources
* [Bios](docs/resources/bios.md)
* [Boot order](docs/resources/boot_order.md)
* [Boot source override](docs/resources/boot_source_override.md)
* [iRMC reset](docs/resources/irmc_reset.md)
* [Power](docs/resources/power.md)
* [Simple update](docs/resources/simple_update.md)
* [Storage volume](docs/resources/storage_volume.md)
* [User account](docs/resources/user_account.md)
* [Virtual media](docs/resources/virtual_media.md)
