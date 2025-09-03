package main

import (
	"log/slog"
	"os/exec"
	"strconv"
)

var gsettingsArgs = []string{"set", "org.gnome.desktop.notifications", "show-banners"}

func setGnomeShellDNDStatus(meetingFound bool) error {
	logger := slog.Default()
	args := []string{}
	args = append(gsettingsArgs, strconv.FormatBool(!meetingFound))
	logger.Debug("Setting Gnome Shell DND status", "cmd", "gsettings", "args", args, "status", meetingFound)
	_, err := exec.Command("gsettings", args...).Output()
	return err
}
