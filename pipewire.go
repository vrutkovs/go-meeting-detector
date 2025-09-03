package main

import (
	"fmt"
	"log/slog"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

const (
	expectedState = "running"
)

// PipeWireClient provides methods to interact with PipeWire with structured logging.
type PipeWireClient struct {
	logger         *slog.Logger
	deviceIDRegexp *regexp.Regexp
	stateRegexp    *regexp.Regexp
}

// NewPipeWireClient creates a new PipeWireClient.
// It compiles the necessary regular expressions and returns a client instance.
// A logger must be provided for structured logging.
func NewPipeWireClient(logger *slog.Logger) (*PipeWireClient, error) {
	if logger == nil {
		logger = slog.Default() // Use default logger if none provided
	}

	deviceIDRegexp, err := regexp.Compile(`id ([0-9]+)`)
	if err != nil {
		logger.Error("failed to compile device ID regex", "error", err)
		return nil, fmt.Errorf("failed to compile device ID regex: %w", err)
	}
	stateRegexp, err := regexp.Compile(`state: ([a-z]+)`)
	if err != nil {
		logger.Error("failed to compile state regex", "error", err)
		return nil, fmt.Errorf("failed to compile state regex: %w", err)
	}

	logger.Debug("PipeWireClient initialized successfully")
	return &PipeWireClient{
		logger:         logger,
		deviceIDRegexp: deviceIDRegexp,
		stateRegexp:    stateRegexp,
	}, nil
}

// findDeviceIDByName executes `pw-cli ls` and parses the output to find the device ID
// associated with the given node name.
func (c *PipeWireClient) findDeviceIDByName(nodeName string) (int, error) {
	c.logger.Debug("Attempting to find PipeWire device by name", "nodeName", nodeName)

	cmd := exec.Command("pw-cli", "ls")
	outputBytes, err := cmd.Output()
	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			c.logger.Error("failed to run 'pw-cli ls'", "error", err, "stderr", string(exitError.Stderr))
			return 0, fmt.Errorf("failed to run 'pw-cli ls': %w, stderr: %s", err, exitError.Stderr)
		}
		c.logger.Error("failed to run 'pw-cli ls'", "error", err)
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
			c.logger.Debug("Found node name in pw-cli output", "nodeName", nodeName, "lineIndex", i)
			break
		}
	}

	if !found {
		c.logger.Info("node with name not found", "nodeName", nodeName)
		return 0, fmt.Errorf("node with name '%s' not found in PipeWire output", nodeName)
	}

	// Extract the device ID using the pre-compiled regex.
	regexpMatch := c.deviceIDRegexp.FindStringSubmatch(strings.Join(relevantOutput, "\n"))
	if len(regexpMatch) < 2 {
		c.logger.Error("failed to extract device ID", "nodeName", nodeName, "output", strings.Join(relevantOutput, "\n"))
		return 0, fmt.Errorf("failed to extract device ID for node '%s' from PipeWire output", nodeName)
	}

	strDeviceID := regexpMatch[1]
	deviceID, err := strconv.Atoi(strDeviceID)
	if err != nil {
		c.logger.Error("error converting device ID to integer", "strDeviceID", strDeviceID, "error", err)
		return 0, fmt.Errorf("error converting extracted device ID '%s' to integer: %w", strDeviceID, err)
	}

	c.logger.Debug("Successfully found PipeWire device ID", "nodeName", nodeName, "deviceID", deviceID)
	return deviceID, nil
}

// checkDeviceStatus checks if a PipeWire device is in the expected state ("running").
// It executes `pw-cli i <deviceID>` and parses the output for the device's state.
func (c *PipeWireClient) checkDeviceStatus(deviceID int) (bool, error) {
	c.logger.Debug("Checking PipeWire device status", "deviceID", deviceID, "expectedState", expectedState)

	cmd := exec.Command("pw-cli", "i", strconv.Itoa(deviceID))
	out, err := cmd.Output()
	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			c.logger.Error("failed to run 'pw-cli i'", "deviceID", deviceID, "error", err, "stderr", string(exitError.Stderr))
			return false, fmt.Errorf("failed to run 'pw-cli i %d': %w, stderr: %s", deviceID, err, exitError.Stderr)
		}
		c.logger.Error("failed to run 'pw-cli i'", "deviceID", deviceID, "error", err)
		return false, fmt.Errorf("failed to run 'pw-cli i %d': %w", deviceID, err)
	}

	// Extract the state using the pre-compiled regex.
	regexpMatch := c.stateRegexp.FindStringSubmatch(string(out))
	if len(regexpMatch) < 2 {
		c.logger.Error("failed to find state for device ID in output", "deviceID", deviceID, "output", string(out))
		return false, fmt.Errorf("failed to find state for device ID %d in PipeWire output", deviceID)
	}

	state := regexpMatch[1]
	if state == expectedState {
		c.logger.Debug("PipeWire device is in expected state", "deviceID", deviceID, "currentState", state, "expectedState", expectedState)
		return true, nil
	}

	c.logger.Debug("PipeWire device is NOT in expected state", "deviceID", deviceID, "currentState", state, "expectedState", expectedState)
	return false, nil
}

// findPipeWireDeviceByName finds the PipeWire device ID by its node name.
// This is the public interface function.
func findPipeWireDeviceByName(nodeName string) (int, error) {
	// A default logger can be used here or a more specific one passed down from main
	pc, err := NewPipeWireClient(slog.Default())
	if err != nil {
		slog.Error("failed to initialize PipeWire client", "error", err)
		return 0, err
	}
	return pc.findDeviceIDByName(nodeName)
}

// checkPipeWireDeviceStatus checks if a PipeWire device is in the expected state.
// This is the public interface function.
func checkPipeWireDeviceStatus(deviceID int) bool {
	// A default logger can be used here or a more specific one passed down from main
	pc, err := NewPipeWireClient(slog.Default())
	if err != nil {
		slog.Error("error creating PipeWireClient", "error", err)
		return false
	}
	status, err := pc.checkDeviceStatus(deviceID)
	if err != nil {
		slog.Error("error checking PipeWire device status", "deviceID", deviceID, "error", err)
		return false
	}
	return status
}
