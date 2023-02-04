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
	expectedState  = "running"
	cmdIDFormat    = "pw-cli ls | grep -B10 '%s'"
	cmdStateFormat = "pw-cli i %d | grep state"
)

var gsettingsArgs = []string{"set", "org.gnome.desktop.notifications", "show-banners"}

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
		args := []string{}
		copy(gsettingsArgs, args)
		args = append(args, strconv.FormatBool(!meetingFound))
		_, err := exec.Command("gsettings", args...).Output()
		return err
	}
	return nil
}

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
