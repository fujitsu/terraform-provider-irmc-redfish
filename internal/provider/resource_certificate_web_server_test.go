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
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

const (
	CERT_PUB_KEY_FILE_PATH  = "path/to/cert/pub_key.pub"
	CERT_PRIV_KEY_FILE_PATH = "path/to/cert/priv_key.pub"
)

func TestAccCertificateWebServer_correct(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccCertificateWebServerConfig(creds, CERT_PUB_KEY_FILE_PATH, CERT_PRIV_KEY_FILE_PATH),
			},
		},
	})
}

func TestAccCertificateWebServer_wrong_PathToFile_PubKey(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccCertificateWebServerConfig(creds, "", CERT_PRIV_KEY_FILE_PATH),
				ExpectError: regexp.MustCompile("Failed to upload public certificate"),
			},
		},
	})
}

func TestAccCertificateWebServer_wrong_PathToFile_PrivKey(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccCertificateWebServerConfig(creds, CERT_PUB_KEY_FILE_PATH, ""),
				ExpectError: regexp.MustCompile("Failed to upload private certificate"),
			},
		},
	})
}

func testAccCertificateWebServerConfig(testingInfo TestingServerCredentials, certificateFilePublicKey, certificateTextPrivateKey string) string {
	return fmt.Sprintf(`
	resource  "irmc-redfish_certificate_web_server" "cert_web_server"  {
		server {
			username     = "%s"
			password     = "%s"
			endpoint     = "https://%s"
			ssl_insecure = true
		}
		cert_public_key = "%s"
		cert_private_key = "%s"
	}
	`,
		testingInfo.Username,
		testingInfo.Password,
		testingInfo.Endpoint,
		certificateFilePublicKey,
		certificateTextPrivateKey,
	)
}
