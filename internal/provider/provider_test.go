// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/joho/godotenv"
	"github.com/stmcginnis/gofish"
	"github.com/stmcginnis/gofish/redfish"
)

var (
	creds TestingServerCredentials
)

type TestingServerCredentials struct {
	Username string
	Password string
	Endpoint string
	Insecure bool
}

// testAccProtoV6ProviderFactories are used to instantiate a provider during
// acceptance testing. The factory function will be invoked for every Terraform
// CLI command executed to create a provider server to which the CLI can
// reattach.
var testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"irmc-redfish": providerserver.NewProtocol6WithError(New("test")()),
}

func testAccPreCheck(t *testing.T) {
	// You can add code here to run prior to any test case execution, for example assertions
	// about the appropriate environment variables being set are common to see in a pre-check
	// function.
}

type vmediaType struct {
	MaxDev  int `json:"MaximumNumberOfDevices"`
	FreeDev int `json:"NumberOfFreeDevices"`
}

type vmediaConfig struct {
	Active   bool       `json:"RemoteMountEnabled"`
	CdConfig vmediaType `json:"CDImage"`
	HdConfig vmediaType `json:"HDImage"`
	Etag     string     `json:"@odata.etag"`
}

func testAccPrepareVMediaSlots(creds TestingServerCredentials) {
	clientConfig := gofish.ClientConfig{
		Endpoint:  "https://" + creds.Endpoint,
		Username:  creds.Username,
		Password:  creds.Password,
		BasicAuth: true,
		Insecure:  true,
	}

	api, err := gofish.Connect(clientConfig)
	if err != nil {
		log.Printf("Connect to %s reported error %s", clientConfig.Endpoint, err.Error())
		return
	}

	path := "/redfish/v1/Systems/0/Oem/ts_fujitsu/VirtualMedia"
	resp, err := api.Get(path)
	if err != nil {
		log.Printf("GET on %s reported error %s", path, err.Error())
		return
	}

	if resp.StatusCode == 200 {
		bodyBytes, err := io.ReadAll(resp.Body)
		if err != nil {
			return
		}

		var config vmediaConfig

		err = json.Unmarshal(bodyBytes, &config)
		if err != nil {
			log.Printf("Error while converting body to json (unmarshalling)")
			return
		}

		if config.CdConfig.MaxDev < 2 || config.HdConfig.MaxDev < 2 {
			config.CdConfig.MaxDev = 4
			config.HdConfig.MaxDev = 4

			headers := map[string]string{"If-Match": config.Etag}
			resp, err := api.PatchWithHeaders(path, config, headers)
			if err != nil {
				log.Printf("PATCH on %s reported error '%s'", path, err.Error())
				return
			}

			if resp.StatusCode == 200 {
				log.Print("Vmedia slot number settings changed successfully, 10s timeout will follow now")
				time.Sleep(10 * time.Second)
			}
		}
	}
}

func testAccPrepareStorageVolume(creds TestingServerCredentials) {
	clientConfig := gofish.ClientConfig{
		Endpoint:  "https://" + creds.Endpoint,
		Username:  creds.Username,
		Password:  creds.Password,
		BasicAuth: true,
		Insecure:  true,
	}

	api, err := gofish.Connect(clientConfig)
	if err != nil {
		log.Printf("Connect to %s reported error %s", clientConfig.Endpoint, err.Error())
		return
	}

	system, err := GetSystemResource(api.Service)
	if err != nil {
		log.Printf("Error while getting Systems/0 resource %s", err.Error())
		return
	}

	list_of_storage_controllers, err := system.Storage()
	if err != nil {
		log.Printf("Error while getting list of storage controllers %s", err.Error())
		return
	}

	if len(list_of_storage_controllers) == 0 {
		log.Printf("System does not show any attached storage controller")
		return
	}

	if system.PowerState != redfish.OnPowerState {
		log.Printf("System host is not powered on")
		return
	}
}

func testChangePowerHostState(creds TestingServerCredentials, poweredOn bool) {
	clientConfig := gofish.ClientConfig{
		Endpoint:  "https://" + creds.Endpoint,
		Username:  creds.Username,
		Password:  creds.Password,
		BasicAuth: true,
		Insecure:  true,
	}

	api, err := gofish.Connect(clientConfig)
	if err != nil {
		log.Printf("Connect to %s reported error %s", clientConfig.Endpoint, err.Error())
		return
	}
	if err = changePowerState(api.Service, poweredOn, 100); err != nil {
		log.Printf("Could not change power state %s", err.Error())
	}
}

func init() {
	err := godotenv.Load("redfish_test.env")
	if err != nil {
		fmt.Println(err.Error())
	}

	creds = TestingServerCredentials{
		Username: os.Getenv("TF_TESTING_USERNAME"),
		Password: os.Getenv("TF_TESTING_PASSWORD"),
		Endpoint: os.Getenv("TF_TESTING_ENDPOINT"),
		Insecure: false,
	}
}
