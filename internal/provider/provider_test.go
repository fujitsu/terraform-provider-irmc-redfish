// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/joho/godotenv"
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

func init () {
    err := godotenv.Load("redfish_test.env")
    if err != nil {
        fmt.Println(err.Error())
    }

    creds = TestingServerCredentials {
        Username: os.Getenv("TF_TESTING_USERNAME"),
        Password: os.Getenv("TF_TESTING_PASSWORD"),
        Endpoint: os.Getenv("TF_TESTING_ENDPOINT"),
        Insecure: false,
    }
}
