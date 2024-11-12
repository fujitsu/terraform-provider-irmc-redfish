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
	"context"
	"fmt"
	"regexp"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/stmcginnis/gofish"
)

const (
	userResourceName = "irmc-redfish_user_account.ua"
	userID_import    = "3"
	username_import  = "Test_user_import"
)

// Function finds last highest user ID number  from accounts and returns the next available user ID as a string.
func getHighestUserID(testingInfo TestingServerCredentials) string {

	clientConfig := gofish.ClientConfig{
		Endpoint:  "https://" + testingInfo.Endpoint,
		Username:  testingInfo.Username,
		Password:  testingInfo.Password,
		BasicAuth: true,
		Insecure:  true,
	}

	api, err := gofish.Connect(clientConfig)
	if err != nil {
		tflog.Error(context.Background(), "Failed to connect to Redfish API", map[string]interface{}{
			"endpoint": clientConfig.Endpoint,
			"error":    err.Error(),
		})
		return ""
	}
	defer api.Logout()

	accountList, err := GetListOfUserAccounts(api.Service)

	if err != nil {
		tflog.Error(context.Background(), "Failed to retrieve user accounts", map[string]interface{}{
			"error": err.Error(),
		})
		return ""
	}

	highestID := 0

	for _, account := range accountList {

		id, err := strconv.Atoi(account.ID)
		if err != nil {
			continue
		}

		if id > highestID {
			highestID = id
		}
	}
	if highestID >= 15 {
		tflog.Error(context.Background(), "User ID exceeds the maximum allowed value.")
		return ""
	}

	if highestID == 0 {
		tflog.Error(context.Background(), "No valid user IDs found.")
		return ""
	}

	return strconv.Itoa(highestID + 1)
}

func TestAccRedfishUserAccount_basic(t *testing.T) {

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccRedfishResourceUserAccountConfig(
					creds, getHighestUserID(creds), "test_user", "Test_password123!", "Administrator", true, true,
					"Administrator", "Administrator", true, true, true, true, "RemoteManager", false,
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(userResourceName, "user_username", "test_user"),
					resource.TestCheckResourceAttr(userResourceName, "user_role", "Administrator"),
					resource.TestCheckResourceAttr(userResourceName, "user_shell_access", "RemoteManager"),
					resource.TestCheckResourceAttr(userResourceName, "user_alert_chassis_events", "false"),
					resource.TestCheckResourceAttr(userResourceName, "user_enabled", "true"),
					resource.TestCheckResourceAttr(userResourceName, "user_redfish_enabled", "true"),
					resource.TestCheckResourceAttr(userResourceName, "user_account_config_enabled", "true"),
					resource.TestCheckResourceAttr(userResourceName, "user_irmc_settings_config_enabled", "true"),
					resource.TestCheckResourceAttr(userResourceName, "user_video_redirection_enabled", "true"),
					resource.TestCheckResourceAttr(userResourceName, "user_remote_storage_enabled", "true"),
				),
			},
		},
	})
}

func TestAccRedfishUserAccount_multipleusers(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccRedfishResourceUserAccountConfig(
					creds, getHighestUserID(creds), "test_userxxx", "Test_password123!", "Administrator", true, true,
					"Administrator", "Administrator", true, true, true, true, "RemoteManager", false,
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(userResourceName, "user_username", "test_userxxx"),
					resource.TestCheckResourceAttr(userResourceName, "user_role", "Administrator"),
				),
			},
			{
				Config: testAccRedfishResourceUserAccountConfig(
					creds, getHighestUserID(creds), "hello_p", "Test_password123!", "Operator", false, false,
					"Operator", "Operator", false, false, false, false, "None", true,
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(userResourceName, "user_username", "hello_p"),
					resource.TestCheckResourceAttr(userResourceName, "user_role", "Operator"),
					resource.TestCheckResourceAttr(userResourceName, "user_enabled", "false"),
					resource.TestCheckResourceAttr(userResourceName, "user_redfish_enabled", "false"),
				),
			},
		},
	})
}

func TestAccRedfishUserAccount_negative_wrongRole(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccRedfishResourceUserAccountConfig(
					creds, getHighestUserID(creds), "test_user", "Test_password123!", "SuperUser", true, true,
					"Administrator", "Administrator", true, true, true, true, "RemoteManager", false,
				),
				ExpectError: regexp.MustCompile("exit status 1"),
			},
		},
	})
}

func TestAccRedfishUserAccount_negative_wrongpassword(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccRedfishResourceUserAccountConfig(
					creds, getHighestUserID(creds), "test_user", "hel", "Administrator", true, true,
					"Administrator", "Administrator", true, true, true, true, "None", false,
				),
				ExpectError: regexp.MustCompile("exit status 1"),
			},
		},
	})
}

func TestAccRedfishUserAccount_duplicateUsername(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccRedfishResourceUserAccountConfig(
					creds, getHighestUserID(creds), "admin", "Test_password123!", "Operator", true, true, // admin is default user
					"Operator", "Operator", true, true, true, true, "RemoteManager", false,
				),
				ExpectError: regexp.MustCompile("exit status 1"),
			},
		},
	})
}

func TestAccRedfishUserAccount_ImportUserbasic_success(t *testing.T) {

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccRedfishResourceUserAccountConfig(
					creds, userID_import, username_import, "Test_password123!", "Administrator", true, true,
					"Administrator", "Administrator", true, true, true, true, "RemoteManager", false,
				),
				ResourceName: "irmc-redfish_user_account.ua",
				ImportState:  true,
				ImportStateId: fmt.Sprintf(`{"username":"%s","password":"%s","endpoint":"https://%s","ssl_insecure":true,"user_id":"%s","user_username":"%s"}`,
					creds.Username, creds.Password, creds.Endpoint, userID_import, username_import),
				ExpectError: nil,
			},
		},
	})
}

func TestAccRedfishUserAccount_ImportUser_fail(t *testing.T) {

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccRedfishResourceUserAccountConfig(
					creds, "0", username_import, "Test_password123!", "Administrator", true, true,
					"Administrator", "Administrator", true, true, true, true, "RemoteManager", false,
				),
				ResourceName: "irmc-redfish_user_account.ua",
				ImportState:  true,
				ImportStateId: fmt.Sprintf(`{"username":"%s","password":"%s","endpoint":"https://%s","ssl_insecure":true,"user_id":"%s","user_username":"%s"}`,
					creds.Username, creds.Password, creds.Endpoint, "0", username_import),
				ExpectError: regexp.MustCompile("Error reading Redfish user account"),
			},
		},
	})
}

func testAccRedfishResourceUserAccountConfig(
	testingInfo TestingServerCredentials,
	userID string,
	username string,
	password string,
	role string,
	enabled bool,
	redfishEnabled bool,
	lanChannelRole string,
	serialChannelRole string,
	accountConfigEnabled bool,
	irmcSettingsConfigEnabled bool,
	videoRedirectionEnabled bool,
	remoteStorageEnabled bool,
	shellAccess string,
	alertChassisEventsEnabled bool,
) string {
	return fmt.Sprintf(`resource "irmc-redfish_user_account" "ua" {
		server {
			username     = "%s"
			password     = "%s"
			endpoint     = "https://%s"
			ssl_insecure = true
		}
		user_id                        = "%s"
		user_username                  = "%s"
		user_password                  = "%s"
		user_role                      = "%s"
		user_enabled                   = %t
		user_redfish_enabled           = %t
		user_lanchannel_role           = "%s"
		user_serialchannel_role        = "%s"
		user_account_config_enabled    = %t
		user_irmc_settings_config_enabled = %t
		user_video_redirection_enabled = %t
		user_remote_storage_enabled    = %t
		user_shell_access              = "%s"
		user_alert_chassis_events      = %t
	}`,
		testingInfo.Username,
		testingInfo.Password,
		testingInfo.Endpoint,

		userID,
		username,
		password,
		role,
		enabled,
		redfishEnabled,
		lanChannelRole,
		serialChannelRole,
		accountConfigEnabled,
		irmcSettingsConfigEnabled,
		videoRedirectionEnabled,
		remoteStorageEnabled,
		shellAccess,
		alertChassisEventsEnabled,
	)
}
