/*
Copyright (c) 2025 Fsas Technologies Inc., or its subsidiaries. All Rights Reserved.

Licensed under the Mozilla Public License Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://mozilla.org/MPL/2.0/


Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/stmcginnis/gofish"
)

const irmc_attributes_name = "irmc-redfish_irmc_attributes.attr"

func TestAccRedfishIrmcAttributes(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				PreConfig: func() { testChangeCasLoginUri(creds, "abc/def") },
				Config:    testAccRedfishResourceIrmcAttrConfig_correctAttributes(creds, "abc/def/ghi"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(irmc_attributes_name, "id", IRMC_ATTRIBUTES_SETTINGS_ENDPOINT),
					resource.TestCheckResourceAttr(irmc_attributes_name, "attributes.BmcCasLoginUri", "abc/def/ghi"),
				),
			},
		},
	})
}

func getIrmcAttributesImportConfiguration(creds TestingServerCredentials) (string, error) {
	return fmt.Sprintf("{\"username\":\"%s\", \"password\":\"%s\", \"endpoint\":\"https://%s\", \"ssl_insecure\":true}",
		creds.Username, creds.Password, creds.Endpoint), nil
}

func TestAccRedfishIrmcAttributes_import(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `resource "irmc-redfish_irmc_attributes" "attr" {
                }`,
				ResourceName: irmc_attributes_name,
				ImportState:  true,
				ExpectError:  nil,
				ImportStateIdFunc: func(d *terraform.State) (string, error) {
					return getIrmcAttributesImportConfiguration(creds)
				},
			},
		},
	})
}

func TestAccRedfishIrmcAttributes_negative(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccRedfishResourceIrmcAttrConfig_notSupportedValue(creds),
				ExpectError: regexp.MustCompile("RedfishTaskManager: Error. The value YYY for the property"),
			},
			{
				Config:      testAccRedfishResourceIrmcAttrConfig_notSupportedAttribute(creds),
				ExpectError: regexp.MustCompile("Attribute 'XXX' is not supported by the system"),
			},
			{
				PreConfig:   func() { testChangeCasLoginUri(creds, "abc/def") },
				Config:      testAccRedfishResourceIrmcAttrConfig_correctAttributes(creds, "abc/def"),
				ExpectError: regexp.MustCompile("Empty list of valid & different attributes to be applied"),
			},
		},
	})
}

func testAccRedfishResourceIrmcAttrConfig_correctAttributes(testingInfo TestingServerCredentials, loginUri string) string {
	return fmt.Sprintf(`
	resource "irmc-redfish_irmc_attributes" "attr" {

		server {
		  username     = "%s"
		  password     = "%s"
		  endpoint     = "https://%s"
		  ssl_insecure = true
		}

        attributes = {
            "BmcCasLoginUri": "%s"
        }
	  }
	`,
		testingInfo.Username,
		testingInfo.Password,
		testingInfo.Endpoint,
		loginUri,
	)
}

func testAccRedfishResourceIrmcAttrConfig_notSupportedAttribute(testingInfo TestingServerCredentials) string {
	return fmt.Sprintf(`
	resource "irmc-redfish_irmc_attributes" "attr" {

		server {
		  username     = "%s"
		  password     = "%s"
		  endpoint     = "https://%s"
		  ssl_insecure = true
		}

        attributes = {
            "XXX": "YYY"
        }
	  }
	`,
		testingInfo.Username,
		testingInfo.Password,
		testingInfo.Endpoint,
	)
}

func testAccRedfishResourceIrmcAttrConfig_notSupportedValue(testingInfo TestingServerCredentials) string {
	return fmt.Sprintf(`
	resource "irmc-redfish_irmc_attributes" "attr" {

		server {
		  username     = "%s"
		  password     = "%s"
		  endpoint     = "https://%s"
		  ssl_insecure = true
		}

        attributes = {
            "BmcCasPermissionRedfish": "YYY"
        }
	  }
	`,
		testingInfo.Username,
		testingInfo.Password,
		testingInfo.Endpoint,
	)
}

type casConfig struct {
	Etag     string `json:"@odata.etag"`
	LoginUri string `json:"LoginUri"`
}

func testChangeCasLoginUri(creds TestingServerCredentials, loginUri string) {
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

	isFsas, err := IsFsasCheck(context.Background(), api)
	if err != nil {
		log.Printf("Vendor check reported error %s", err.Error())
		return
	}

	var oemKey string
	if isFsas {
		oemKey = FSAS
	} else {
		oemKey = TS_FUJITSU
	}

	path := fmt.Sprintf("/redfish/v1/Managers/iRMC/Oem/%s/iRMCConfiguration/Cas", oemKey)

	resp, err := api.Get(path)
	if err != nil {
		log.Printf("Get request failed: %s", err.Error())
		return
	}

	if resp.StatusCode == 200 {
		bodyBytes, err := io.ReadAll(resp.Body)
		if err != nil {
			log.Printf("Error reading response body")
			return
		}

		var config casConfig
		err = json.Unmarshal(bodyBytes, &config)
		if err != nil {
			log.Printf("Error while converting body to json (unmarshalling)")
		}

		config.LoginUri = loginUri

		// patch method performs marshalling inside!
		headers := map[string]string{"If-Match": config.Etag}
		resp, err := api.PatchWithHeaders(path, config, headers)
		if err != nil {
			log.Printf("error during patch %s", err.Error())
			return
		}

		if resp.StatusCode == 200 {
			log.Print("Finished successfully")
		}
	}
}
