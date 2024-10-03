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
* [Virtual media](docs/resources/virtual_media.md)
