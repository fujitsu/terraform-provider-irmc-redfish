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
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/stmcginnis/gofish"
	"github.com/stmcginnis/gofish/redfish"
)

const (
	BIOS_ENDPOINT = "/redfish/v1/Systems/0/Bios"
)

// isPoweredOn returns information whether host defined by service is powered on or not.
func isPoweredOn(service *gofish.Service) (bool, error) {
	system, err := GetSystemResource(service)
	if err != nil {
		return false, err
	}

	if system.PowerState == redfish.OnPowerState {
		return true, nil
	}

	return false, nil
}

// waitUntilHostStateChanged waits with timeout until expectedPoweredOn will be reached
// by target defined as service.
func waitUntilHostStateChanged(service *gofish.Service, expectedPoweredOn bool, timeout int64) error {
	startTime := time.Now().Unix()
	for {
		poweredOn, err := isPoweredOn(service)
		if err != nil {
			return err
		}

		if expectedPoweredOn {
			if poweredOn {
				return nil
			}
		} else {
			if !poweredOn {
				return nil
			}
		}

		if time.Now().Unix()-startTime > timeout {
			return fmt.Errorf("error. Host state has not been changed within given timeout %d", timeout)
		}

		time.Sleep(2 * time.Second)
	}
}

type tsBiosObject struct {
	IsBiosInPOSTPhase bool `json:"IsBiosInPostPhase"`
}

type biosOemObject struct {
	Ts_fujitsu tsBiosObject `json:"ts_fujitsu"`
}

type biosObject struct {
	Oem biosOemObject `json:"Oem"`
}

// isBiosInPOSTPhase returns information whether host reports
// being in POST state or not.
func isBiosInPOSTPhase(service *gofish.Service) (bool, error) {
	res, err := service.GetClient().Get(BIOS_ENDPOINT)
	if err != nil {
		return false, err
	}

	if res.StatusCode != http.StatusOK {
		return false, fmt.Errorf("Return status code != 200")
	}

	bodyBytes, err := io.ReadAll(res.Body)
	if err != nil {
		return false, err
	}

	var config biosObject
	err = json.Unmarshal(bodyBytes, &config)
	if err != nil {
		return false, err
	}

	return config.Oem.Ts_fujitsu.IsBiosInPOSTPhase, nil
}

// waitUntilHostStateChangedEnhanced waits until host will change its state
// based on BIOS POST phase (exit of the POST phase together with host powered on state
// is treated as reached powered on state).
func waitUntilHostStateChangedEnhanced(service *gofish.Service, expectedPoweredOn bool, timeout int64) error {
	if !expectedPoweredOn {
		return waitUntilHostStateChanged(service, expectedPoweredOn, timeout)
	}

	startTime := time.Now().Unix()
	for {
		// wait until BIOS will report POST state
		for {
			biosDuringPOST, err := isBiosInPOSTPhase(service)
			if err != nil {
				return err
			}

			if biosDuringPOST {
				break
			} else {
				time.Sleep(time.Second)

				if time.Now().Unix()-startTime > timeout {
					return fmt.Errorf("BIOS did not entered POST within given timeout %d", timeout)
				}
			}
		}

		// wait until BIOS will stop report POST state and host will be still on
		for {
			biosDuringPOST, err := isBiosInPOSTPhase(service)
			if err != nil {
				return err
			}

			if !biosDuringPOST {
				isPoweredOn, err := isPoweredOn(service)
				if err != nil {
					return err
				}

				if isPoweredOn {
					return nil
				} else {
					return fmt.Errorf("BIOS exited POST but host powered off")
				}
			} else {
				time.Sleep(2 * time.Second)
			}

			if time.Now().Unix()-startTime > timeout {
				return fmt.Errorf("Operation not finished within given timeout %d", timeout)
			}
		}
	}
}

// changePowerState tries to change host state to value defined in powerOn with timeout
// when requested power state should be reached.
func changePowerState(service *gofish.Service, powerOn bool, timeout int64) error {
	system, err := GetSystemResource(service)
	if err != nil {
		return err
	}

	isPoweredOn, err := isPoweredOn(service)
	if err != nil {
		return err
	}

	operation := redfish.OnResetType
	expectedTargetState := true
	if powerOn {
		if isPoweredOn {
			return nil
		}
	} else {
		if !isPoweredOn {
			return nil
		} else {
			operation = redfish.ForceOffResetType
			expectedTargetState = false
		}
	}

	err = system.Reset(operation)
	if err != nil {
		return err
	}

	err = waitUntilHostStateChangedEnhanced(service, expectedTargetState, timeout)
	if err != nil {
		return err
	}

	return nil
}

// resetHost calls host reset using resetType defined by caller.
func resetHost(service *gofish.Service, resetType redfish.ResetType, timeout int64) error {
	system, err := GetSystemResource(service)
	if err != nil {
		return err
	}

	err = system.Reset(resetType)
	if err != nil {
		return err
	}

	expectedTargetState := true
	if resetType == redfish.GracefulShutdownResetType || resetType == redfish.PushPowerButtonResetType {
		// Assumption: host is powered on if someone requested reset
		expectedTargetState = false
	}

	err = waitUntilHostStateChangedEnhanced(service, expectedTargetState, timeout)
	if err != nil {
		return err
	}

	return nil
}

// resetOrPowerOnHostWithPostCheck powers on host if it's currently powered off
// or performs requested resetType operation if host is on within given timeout.
func resetOrPowerOnHostWithPostCheck(service *gofish.Service, resetType redfish.ResetType, timeout int64) error {
	poweredOn, err := isPoweredOn(service)
	if err != nil {
		return err
	}

	if !poweredOn {
		if err = changePowerState(service, true, timeout); err != nil {
			return err
		}
	} else {
		if err = resetHost(service, resetType, timeout); err != nil {
			return err
		}
	}

	return nil
}
