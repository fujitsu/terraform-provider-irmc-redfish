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
	CERT_FILE_PATH = "/path/to/certificate"
	CERT_TEXT      = `-----BEGIN CERTIFICATE-----
your correct cert
-----END CERTIFICATE-----`
	CERT_TEXT_FAIL = `-----BEGIN CERTIFICATE-----
invalid_certificate
-----END CERTIFICATE-----`
)

func TestAccCertificateCaUpdDeployResource_correct_File(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccTestAccCertificateCaUpdDeployResourceConfig(creds, "File", CERT_FILE_PATH, ""),
			},
		},
	})
}

func TestAccCertificateCaUpdDeployResource_correct_Text(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccTestAccCertificateCaUpdDeployResourceConfig(creds, "Text", "", CERT_TEXT),
			},
		},
	})
}

func TestAccCertificateCaUpdDeployResource_wrong_sameCert(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccTestAccCertificateCaUpdDeployResourceConfig(creds, "Text", "", CERT_TEXT_FAIL),
				ExpectError: regexp.MustCompile("Text Certificate Upload failed."),
			},
		},
	})
}

func TestAccCertificateCaUpdDeployResource_wrong_PathToFile(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccTestAccCertificateCaUpdDeployResourceConfig(creds, "File", "/home/polecp/terraform/test.pem", ""),
				ExpectError: regexp.MustCompile("File Certificate Upload failed."),
			},
		},
	})
}

func testAccTestAccCertificateCaUpdDeployResourceConfig(testingInfo TestingServerCredentials, certificateUploadType, certificateFile, certificateText string) string {
	return fmt.Sprintf(`
	resource "irmc-redfish_certificate_ca_upd_deploy" "ca_upd_deploy" {
		server {
			username     = "%s"
			password     = "%s"
			endpoint     = "https://%s"
			ssl_insecure = true
		}

		certificate_upload_type  = "%s"
		certificate_file     = "%s"
		certificate_text     = <<EOT
%s
EOT
	}
	`,
		testingInfo.Username,
		testingInfo.Password,
		testingInfo.Endpoint,
		certificateUploadType,
		certificateFile,
		certificateText,
	)
}
