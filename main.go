package main

import (
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	logger := slog.Default() // Using default logger for main, could be configured further.

	// Read env vars
	mqttHost, ok := os.LookupEnv("MQTT_HOST")
	if !ok {
		logger.Error("environment variable unset", "var", "MQTT_HOST")
		os.Exit(1)
	}
	mqttPort, ok := os.LookupEnv("MQTT_PORT")
	if !ok {
		logger.Error("environment variable unset", "var", "MQTT_PORT")
		os.Exit(1)
	}
	mqttUsername, ok := os.LookupEnv("MQTT_USER")
	if !ok {
		logger.Error("environment variable unset", "var", "MQTT_USER")
		os.Exit(1)
	}
	mqttPassword, ok := os.LookupEnv("MQTT_PASSWORD")
	if !ok {
		logger.Error("environment variable unset", "var", "MQTT_PASSWORD")
		os.Exit(1)
	}
	mqttTopic, ok := os.LookupEnv("MQTT_TOPIC")
	if !ok {
		logger.Error("environment variable unset", "var", "MQTT_TOPIC")
		os.Exit(1)
	}
	nodeName, ok := os.LookupEnv("PW_NODE_NAME")
	if !ok {
		logger.Error("environment variable unset", "var", "PW_NODE_NAME")
		os.Exit(1)
	}

	logger.Info("environment variables loaded",
		"mqttHost", mqttHost,
		"mqttPort", mqttPort,
		"mqttUsername", mqttUsername,
		"mqttTopic", mqttTopic,
		"pipewireNodeName", nodeName)

	// Make an mqtt client
	logger.Info("connecting to MQTT broker",
		"host", mqttHost,
		"port", mqttPort,
		"username", mqttUsername)
	mqtt := NewMqtt(mqttHost, mqttPort, mqttUsername, mqttPassword)
	if token := mqtt.client.Connect(); token.Wait() && token.Error() != nil {
		logger.Error("failed to connect to MQTT broker", "error", token.Error())
		os.Exit(1)
	}
	logger.Info("successfully connected to MQTT broker")

	// Check for meeting every second
	ticker := time.NewTicker(time.Second)
	quit := make(chan struct{})

	// Close on SIGTERM
	c := make(chan os.Signal, 2)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		logger.Info("received shutdown signal, initiating graceful shutdown")
		close(quit)
		// Give some time for cleanup, then exit
		time.Sleep(500 * time.Millisecond)
		os.Exit(0)
	}()

	logger.Info("waiting for PipeWire events...")

	for {
		select {
		case <-ticker.C:
			logger.Debug("ticker ticked, checking PipeWire device status")
			deviceID, err := findPipeWireDeviceByName(nodeName)
			if err != nil {
				logger.Error("failed to find PipeWire device", "nodeName", nodeName, "error", err)
				// Continue to next tick even if device not found to retry
				continue
			}

			// checkPipeWireDeviceStatus now handles its own logging for status results
			meetingFound := checkPipeWireDeviceStatus(deviceID)

			if err := toggleMode(mqtt, mqttTopic, meetingFound); err != nil {
				logger.Error("failed to toggle mode or set DND status", "meetingFound", meetingFound, "error", err)
			}
		case <-quit:
			logger.Info("disconnecting from MQTT broker")
			mqtt.client.Disconnect(250)
			ticker.Stop()
			logger.Info("application stopped gracefully")
			return
		}
	}
}

func toggleMode(mqtt *Mqtt, topic string, meetingFound bool) error {
	logger := slog.Default() // Or pass a logger from main
	if mqtt.State != meetingFound {
		logger.Info("meeting status changed, toggling mode", "oldState", mqtt.State, "newState", meetingFound)
		mqtt.setState(topic, meetingFound)
		if err := setGnomeShellDNDStatus(meetingFound); err != nil {
			logger.Error("failed to set Gnome Shell DND status", "meetingFound", meetingFound, "error", err)
			return fmt.Errorf("failed to set Gnome Shell DND status: %w", err)
		}
		logger.Info("Gnome Shell DND status set successfully", "status", meetingFound)
	} else {
		logger.Debug("meeting status unchanged, no action taken", "currentState", mqtt.State)
	}
	return nil
}
