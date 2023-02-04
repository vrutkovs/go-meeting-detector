package main

import (
	"os/exec"
	"strconv"
)

var gsettingsArgs = []string{"set", "org.gnome.desktop.notifications", "show-banners"}

func setGnomeShellDNDStatus(meetingFound bool) error {
	args := []string{}
	copy(gsettingsArgs, args)
	args = append(args, strconv.FormatBool(!meetingFound))
	_, err := exec.Command("gsettings", args...).Output()
	return err
}
