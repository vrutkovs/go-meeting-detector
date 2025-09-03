package main

import (
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

const (
	expectedState = "running"
)

// PipeWireClient provides methods to interact with PipeWire.
type PipeWireClient struct {
	deviceIDRegexp *regexp.Regexp
	stateRegexp    *regexp.Regexp
}

// NewPipeWireClient creates a new PipeWireClient.
// It compiles the necessary regular expressions and returns a client instance.
func NewPipeWireClient() (*PipeWireClient, error) {
	deviceIDRegexp, err := regexp.Compile(`id ([0-9]+)`)
	if err != nil {
		return nil, fmt.Errorf("failed to compile device ID regex: %w", err)
	}
	stateRegexp, err := regexp.Compile(`state: ([a-z]+)`)
	if err != nil {
		return nil, fmt.Errorf("failed to compile state regex: %w", err)
	}
	return &PipeWireClient{
		deviceIDRegexp: deviceIDRegexp,
		stateRegexp:    stateRegexp,
	}, nil
}

// findDeviceIDByName executes `pw-cli ls` and parses the output to find the device ID
// associated with the given node name.
func (c *PipeWireClient) findDeviceIDByName(nodeName string) (int, error) {
	cmd := exec.Command("pw-cli", "ls")
	outputBytes, err := cmd.Output()
	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			return 0, fmt.Errorf("failed to run 'pw-cli ls': %w, stderr: %s", err, exitError.Stderr)
		}
		return 0, fmt.Errorf("failed to run 'pw-cli ls': %w", err)
	}

	// Simulate `grep -B10 'nodeName'` by splitting the output into lines
	// and looking for the nodeName, then capturing preceding lines.
	lines := strings.Split(string(outputBytes), "\n")
	var relevantOutput []string
	found := false
	for i := 0; i < len(lines); i++ {
		if strings.Contains(lines[i], nodeName) {
			found = true
			// Include up to 10 preceding lines and the current line to ensure we capture the 'id' line.
			start := i - 10
			if start < 0 {
				start = 0
			}
			relevantOutput = append(relevantOutput, lines[start:i+1]...)
			break
		}
	}

	if !found {
		return 0, fmt.Errorf("node with name '%s' not found in PipeWire output", nodeName)
	}

	// Extract the device ID using the pre-compiled regex.
	regexpMatch := c.deviceIDRegexp.FindStringSubmatch(strings.Join(relevantOutput, "\n"))
	if len(regexpMatch) < 2 {
		return 0, fmt.Errorf("failed to extract device ID for node '%s' from PipeWire output", nodeName)
	}

	strDeviceID := regexpMatch[1]
	deviceID, err := strconv.Atoi(strDeviceID)
	if err != nil {
		return 0, fmt.Errorf("error converting extracted device ID '%s' to integer: %w", strDeviceID, err)
	}
	return deviceID, nil
}

// checkDeviceStatus checks if a PipeWire device is in the expected state ("running").
// It executes `pw-cli i <deviceID>` and parses the output for the device's state.
func (c *PipeWireClient) checkDeviceStatus(deviceID int) (bool, error) {
	cmd := exec.Command("pw-cli", "i", strconv.Itoa(deviceID))
	out, err := cmd.Output()
	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			return false, fmt.Errorf("failed to run 'pw-cli i %d': %w, stderr: %s", deviceID, err, exitError.Stderr)
		}
		return false, fmt.Errorf("failed to run 'pw-cli i %d': %w", deviceID, err)
	}

	// Extract the state using the pre-compiled regex.
	regexpMatch := c.stateRegexp.FindStringSubmatch(string(out))
	if len(regexpMatch) < 2 {
		return false, fmt.Errorf("failed to find state for device ID %d in PipeWire output", deviceID)
	}

	state := regexpMatch[1]
	return state == expectedState, nil
}

// findPipeWireDeviceByName finds the PipeWire device ID by its node name.
// This is the public interface function.
func findPipeWireDeviceByName(nodeName string) (int, error) {
	pc, err := NewPipeWireClient()
	if err != nil {
		return 0, err
	}
	return pc.findDeviceIDByName(nodeName)
}

// checkPipeWireDeviceStatus checks if a PipeWire device is in the expected state.
// This is the public interface function.
func checkPipeWireDeviceStatus(deviceID int) bool {
	pc, err := NewPipeWireClient()
	if err != nil {
		fmt.Printf("Error creating PipeWireClient: %v\n", err)
		return false
	}
	status, err := pc.checkDeviceStatus(deviceID)
	if err != nil {
		fmt.Printf("Error checking PipeWire device status for ID %d: %v\n", deviceID, err)
		return false
	}
	return status
}
