package main

import (
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
)

const (
	expectedState  = "running"
	cmdIDFormat    = "pw-cli ls | grep -B10 '%s'"
	cmdStateFormat = "pw-cli i %d | grep state"
)

func findPipeWireDeviceByName(nodeName string, deviceIDRegexp *regexp.Regexp) (int, error) {
	cmd := fmt.Sprintf(cmdIDFormat, nodeName)
	out, err := exec.Command("/bin/sh", "-c", cmd).Output()
	if err != nil {
		return 0, fmt.Errorf("failed to run command to find device ID by node name")
	}
	regexpMatch := deviceIDRegexp.FindStringSubmatch(string(out))
	if len(regexpMatch) < 2 {
		return 0, fmt.Errorf("failed to find device ID by node name in output")
	}
	strDeviceID := regexpMatch[1]
	deviceID, err := strconv.Atoi(strDeviceID)
	if err != nil {
		return 0, fmt.Errorf("error converting device ID: %v", err)
	}
	return deviceID, nil

}

func checkPipeWireDeviceStatus(deviceID int, stateRegexp *regexp.Regexp) bool {
	out, err := exec.Command("/bin/sh", "-c", fmt.Sprintf(cmdStateFormat, deviceID)).Output()
	if err != nil {
		return false
	}
	regexpMatch := stateRegexp.FindStringSubmatch(string(out))
	if len(regexpMatch) < 2 {
		return false
	}
	state := regexpMatch[1]
	return state == expectedState

}
