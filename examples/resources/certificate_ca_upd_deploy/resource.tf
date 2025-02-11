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

resource "irmc-redfish_certificate_ca_upd_deploy" "ca_upd_deploy" {
  for_each = var.rack1
  server {
    username     = each.value.username
    password     = each.value.password
    endpoint     = each.value.endpoint
    ssl_insecure = each.value.ssl_insecure
  }
  certificate_upload_type = "Text"
  certificate_file        = "/path/to/certificate/file.pem"
  certificate_text        = <<EOF
-----BEGIN CERTIFICATE-----
MIIEXTCCAsWgAwIBAgIRAKfUp6F8dkqaWvKTa+RdUCAwDQYJKoZIhvcNAQELBQAw
gZcxHjAcBgNVBAoTFW1rY2VydCBkZXZlbG9wbWVudCBDQTE2MDQGA1UECwwtV0lO
U1JWMjAxMlIyWDY0XEFkbWluaXN0cmF0b3JAV2luU3J2MjAxMlIyeDY0MT0wOwYD
VQQDDDRta2NlcnQgV0lOU1JWMjAxMlIyWDY0XEFkbWluaXN0cmF0b3JAV2luU3J2
MjAxMlIyeDY0MB4XDTIyMDUxNzA5NDMyNVoXDTI0MDgxNzA5NDMyNVowYTEnMCUG
A1UEChMebWtjZXJ0IGRldmVsb3BtZW50IGNlcnRpZmljYXRlMTYwNAYDVQQLDC1X
SU5TUlYyMDEyUjJYNjRcQWRtaW5pc3RyYXRvckBXaW5TcnYyMDEyUjJ4NjQwggEi
MA0GCSqGSIb3DQEBAQUAA4IBDwAwggEKAoIBAQDVI+fogvt+UCOtTl7MJd4602xN
yfGCviioF1FFUDrxZSGB0gBuiVVUCvOE+6NFEDCIvlpAJ0TpcrK2sHy21SvqGjCP
LIARa+aAUNIN5drkvXHlmlhEJ1/bQAxi5c89OKjRT5c5vhr+JgGZcW7/1vPkBhO0
e1ChrUK1q4/K3sBkR0HmYvCYPq/orEB5917T21Brt8z10hA3gNA6UbLWvYiqebaM
wL25pY8KhpMXrei6uEOg4zYuzwZ40HqgkvBPQIHzbsjDGIbiogPCw8W1tBzPLHYY
svaHbDpFjJxButyrBH2kn4anZjRF3CkRznVHgZPr4A6W/mWS55MD3KSih47nAgMB
AAGjWTBXMA4GA1UdDwEB/wQEAwIFoDATBgNVHSUEDDAKBggrBgEFBQcDATAfBgNV
HSMEGDAWgBQcUy5uFJP/6h7ak29iJKvSLZwdhTAPBgNVHREECDAGhwSsEUgVMA0G
CSqGSIb3DQEBCwUAA4IBgQBO/ectvVnjqs3PeMfR3+gcXznJI/DlFLb+7Fwhd2Pf
yknGQ29kGnh9gOyAnS+4p8ltNE7shCiua1oC5/649Ni1R6YbsFW577yofXE1kXEg
KoI1qD+/UVPeOU4gljlofq91CJA3hYqZJb19q/rRxEkHJ+S969isRDKIb0Dqy8Kw
9mlukJLQVNQNsTYd5mL+++6qAocXnYsk4FUbp58eStw+kVIB2R6/kFLWGkp/G53j
pIxLjCHYskZnSsUoolgEeM+/WaTvXJLWZ0lmLSLg8YRFII4/iDJcHHV4Gm1iUtP0
3B6dBygXwt0TUNoH7riihEZrco2l/9MU3cyOXmb98o0l3pUXP4fADJnqMrXf3ANT
azKPKylpD0qOYP9RnJuddbFNPR4NEKsNTEGmYjwoC0C8QK6u+f5xDOgsEg46VCLI
kQrCnrc7BNLL+kaLKLwfymaT6/OvLWbIPetIxSeyWdXCP8KFp6jENZJAh3g9CcNO
y5BJYJg9pVi/gvyIEgtqe6s=
-----END CERTIFICATE-----
  EOF
}
