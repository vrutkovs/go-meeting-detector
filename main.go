package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	// Read env vars
	mqttHost, ok := os.LookupEnv("MQTT_HOST")
	if !ok {
		panic("MQTT_HOST unset")
	}
	mqttPort, ok := os.LookupEnv("MQTT_PORT")
	if !ok {
		panic("MQTT_PORT unset")
	}
	mqttUsername, ok := os.LookupEnv("MQTT_USER")
	if !ok {
		panic("MQTT_USER unset")
	}
	mqttPassword, ok := os.LookupEnv("MQTT_PASSWORD")
	if !ok {
		panic("MQTT_PASSWORD unset")
	}
	mqttTopic, ok := os.LookupEnv("MQTT_TOPIC")
	if !ok {
		panic("MQTT_TOPIC unset")
	}
	nodeName, ok := os.LookupEnv("PW_NODE_NAME")
	if !ok {
		panic("PW_NODE_NAME unset")
	}
	// Make an mqtt client
	fmt.Printf("Connecting to tcp://%s:%s as %s\n", mqttHost, mqttPort, mqttUsername)
	mqtt := NewMqtt(mqttHost, mqttPort, mqttUsername, mqttPassword)
	if token := mqtt.client.Connect(); token.Wait() && token.Error() != nil {
		panic(token.Error())
	}

	// Check for meeting every second
	ticker := time.NewTicker(time.Second)
	quit := make(chan struct{})

	// Close on SIGTERM
	c := make(chan os.Signal, 2)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		close(quit)
		os.Exit(1)
	}()

	fmt.Printf("Waiting for Pipewire events")

	for {
		select {
		case <-ticker.C:
			deviceID, err := findPipeWireDeviceByName(nodeName)
			if err != nil {
				print(err)
			}
			meetingFound := checkPipeWireDeviceStatus(deviceID)
			if err := toggleMode(mqtt, mqttTopic, meetingFound); err != nil {
				print(err)
			}
		case <-quit:
			mqtt.client.Disconnect(250)
			ticker.Stop()
			return
		}
	}
}

func toggleMode(mqtt *Mqtt, topic string, meetingFound bool) error {
	if mqtt.State != meetingFound {
		mqtt.setState(topic, meetingFound)
		return setGnomeShellDNDStatus(meetingFound)
	}
	return nil
}
