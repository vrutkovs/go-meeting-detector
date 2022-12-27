package main

import (
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"regexp"
	"strconv"
	"syscall"
	"time"
)

const (
	expectedState = "running"
	cmdFormat     = "pw-cli i %d | grep state"
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
	strDeviceID, ok := os.LookupEnv("PW_DEVICE_ID")
	if !ok {
		panic("PW_DEVICE_ID unset")
	}
	deviceID, err := strconv.Atoi(strDeviceID)
	if err != nil {
		panic(fmt.Sprintf("Error converting device ID: %v", err))
	}

	// Make an mqtt client
	fmt.Printf("Connecting to tcp://%s:%s as %s", mqttHost, mqttPort, mqttUsername)
	mqtt := NewMqtt(mqttHost, mqttPort, mqttUsername, mqttPassword)
	if token := mqtt.client.Connect(); token.Wait() && token.Error() != nil {
		panic(token.Error())
	}

	// Prepare meeting regexp
	stateRegexp := regexp.MustCompile(`.*"(.+)".*`)

	// Check for meeting every second
	ticker := time.NewTicker(time.Second)
	quit := make(chan struct{})

	// Close on SIGTERM
	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		close(quit)
		os.Exit(1)
	}()

	for {
		select {
		case <-ticker.C:
			meetingFound := checkPipeWireDeviceStatus(deviceID, stateRegexp)
			if mqtt.State != meetingFound {
				mqtt.setState(meetingFound)
			}
		case <-quit:
			mqtt.client.Disconnect(250)
			ticker.Stop()
			return
		}
	}
}

func checkPipeWireDeviceStatus(deviceID int, stateRegexp *regexp.Regexp) bool {
	out, err := exec.Command("/bin/sh", "-c", fmt.Sprintf(cmdFormat, deviceID)).Output()
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
