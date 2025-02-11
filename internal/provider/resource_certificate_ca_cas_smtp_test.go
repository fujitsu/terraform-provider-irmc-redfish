/*
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
*/

package provider

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

const (
	CERT_CA_CAS_SMTP_FILE_PATH = "path/to/cert/cert.pem"
)

func TestAccCertificateCaCasSmtp_correct(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccCertificateCaCasSmtpConfig(creds, CERT_CA_CAS_SMTP_FILE_PATH),
			},
		},
	})
}

func TestAccCertificateCaCasSmtp_wrong_PathToFile(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccCertificateCaCasSmtpConfig(creds, ""),
				ExpectError: regexp.MustCompile("Failed to upload public certificate"),
			},
		},
	})
}

func testAccCertificateCaCasSmtpConfig(testingInfo TestingServerCredentials, certificateCaFile string) string {
	return fmt.Sprintf(`
	resource  "irmc-redfish_certificate_ca_cas_smtp" "ca_cas_smtp" {
		server {
			username     = "%s"
			password     = "%s"
			endpoint     = "https://%s"
			ssl_insecure = true
		}
		certificate_ca_file = "%s"
	}
	`,
		testingInfo.Username,
		testingInfo.Password,
		testingInfo.Endpoint,
		certificateCaFile,
	)
}
