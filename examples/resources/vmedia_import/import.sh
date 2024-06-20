#! /bin/bash

terraform import irmc-redfish_virtual_media.vm '{"id":"/redfish/v1/Managers/iRMC/VirtualMedia/0", "username": "admin", "password":"admin", "endpoint":"https://10.172.201.188", "ssl_insecure": true}'
