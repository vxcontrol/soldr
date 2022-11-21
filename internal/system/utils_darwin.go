//go:build darwin
// +build darwin

package system

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
	"sync"
)

var execLock sync.Mutex

func getOSName() string {
	value, err := execCmd("sw_vers", "-productName")
	if err != nil {
		value = ""
	}
	value = strings.Replace(value, "\r", ``, -1)
	value = strings.Replace(value, "\n", ``, -1)
	value = strings.Trim(value, " ")
	return value
}

func getOSVer() string {
	value, err := execCmd("sw_vers", "-productVersion")
	if err != nil {
		value = "0"
	}
	value = strings.Replace(value, "\r", ``, -1)
	value = strings.Replace(value, "\n", ``, -1)
	value = strings.Trim(value, " ")
	return value
}

func getMachineID() (string, error) {
	out, err := execCmd("ioreg", "-rd1", "-c", "IOPlatformExpertDevice")
	if err != nil {
		return "", err
	}
	id, err := extractID(out)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(strings.Trim(id, "\n")), nil
}

func extractID(lines string) (string, error) {
	const uuidParamName = "IOPlatformUUID"
	for _, line := range strings.Split(lines, "\n") {
		if strings.Contains(line, uuidParamName) {
			parts := strings.SplitAfter(line, `" = "`)
			if len(parts) == 2 {
				return strings.TrimRight(parts[1], `"`), nil
			}
		}
	}
	return "", fmt.Errorf("failed to extract the '%s' value from the `ioreg` output", uuidParamName)
}

func updateEtcPasswd(entries map[string]etcPasswdEntry) {
	users, err := execCmd("dscl", ".", "list", "/Users")
	if err != nil {
		return
	}
	for _, usr := range strings.Split(users, "\n") {
		if _, ok := entries[usr]; !ok && usr != "" {
			entries[usr] = etcPasswdEntry{username: usr}
		}
	}
}

func execCmd(scmd string, args ...string) (string, error) {
	execLock.Lock()
	defer execLock.Unlock()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd := exec.Command(scmd, args...)
	cmd.Stdin = strings.NewReader("")
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		return "", err
	}

	return stdout.String(), nil
}
