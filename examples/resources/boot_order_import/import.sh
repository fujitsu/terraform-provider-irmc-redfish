#! /bin/bash

terraform import irmc-redfish_boot_order.bo '{"username": "admin", "password":"adminADMIN123", "endpoint":"https://10.172.201.40", "ssl_insecure": true}'
