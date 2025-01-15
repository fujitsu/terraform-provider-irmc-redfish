#! /bin/bash

TF_LOG=INFO terraform import irmc-redfish_storage_volume.vol '{"id":"/redfish/v1/Systems/0/Storage/0/Volumes/239", "username":"admin", "password":"adminADMIN123", "endpoint":"https://10.172.201.40", "ssl_insecure": true}'
