//go:build linux
// +build linux

package system

import (
	"io/ioutil"
	"strings"

	"github.com/acobaugh/osrelease"
)

func getOSName() string {
	osrelease, err := osrelease.Read()
	if err != nil {
		return ""
	}
	return osrelease["NAME"]
}

func getOSVer() string {
	osrelease, err := osrelease.Read()
	if err != nil {
		return "0"
	}
	return osrelease["VERSION_ID"]
}

func updateEtcPasswd(entries map[string]etcPasswdEntry) {}

func getMachineID() (string, error) {
	const (
		// dbusPath is the default path for dbus machine id.
		dbusPath = "/var/lib/dbus/machine-id"
		// dbusPathEtc is the default path for dbus machine id located in /etc.
		// Some systems (like Fedora 20) only know this path.
		// Sometimes it's the other way round.
		dbusPathEtc = "/etc/machine-id"
	)

	id, err := ioutil.ReadFile(dbusPath)
	if err != nil {
		id, err = ioutil.ReadFile(dbusPathEtc)
	}
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(strings.Trim(string(id), "\n")), nil
}
