package main

import (
	"os"
	"os/signal"
	"regexp"
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
	print("Connecting to tcp://%s:%s as %s", mqttHost, mqttPort, mqttUsername)
	mqtt := NewMqtt(mqttHost, mqttPort, mqttUsername, mqttPassword)
	if token := mqtt.client.Connect(); token.Wait() && token.Error() != nil {
		panic(token.Error())
	}

	// Prepare regexps
	stateRegexp := regexp.MustCompile(`.*"(.+)".*`)
	deviceIDRegexp := regexp.MustCompile(`id (\d+), `)

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

	for {
		select {
		case <-ticker.C:
			deviceID, err := findPipeWireDeviceByName(nodeName, deviceIDRegexp)
			if err != nil {
				print(err)
			}
			meetingFound := checkPipeWireDeviceStatus(deviceID, stateRegexp)
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
