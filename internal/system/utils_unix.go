//go:build !windows
// +build !windows

package system

import (
	"fmt"
	"io/ioutil"
	"os/user"
	"strconv"
	"strings"

	"soldr/internal/protoagent"
)

func getUsersInformation() []*protoagent.Information_User {
	users := make([]*protoagent.Information_User, 0)

	entries, err := loadEtcPasswd()
	if err != nil {
		return users
	}

	updateEtcPasswd(entries)
	for _, usr := range entries {
		name := usr.username
		item := &protoagent.Information_User{
			Name:   &name,
			Groups: []string{},
		}

		if strings.HasPrefix(usr.username, "_") {
			continue
		}
		if strings.HasSuffix(usr.shell, "bin/nologin") {
			continue
		}
		if strings.HasSuffix(usr.shell, "bin/false") {
			continue
		}

		us, err := user.Lookup(name)
		if err != nil {
			continue
		}
		gs, err := us.GroupIds()
		if err != nil {
			continue
		}
		for _, g := range gs {
			ug, err := user.LookupGroupId(g)
			if err != nil {
				continue
			}
			if ug.Name != "" && ug.Name != "None" {
				item.Groups = append(item.Groups, ug.Name)
			}
		}

		users = append(users, item)
	}

	return users
}

type etcPasswdEntry struct {
	username string
	password string
	uid      int
	gid      int
	info     string
	homedir  string
	shell    string
}

func loadEtcPasswd() (map[string]etcPasswdEntry, error) {
	content, err := ioutil.ReadFile("/etc/passwd")
	if err != nil {
		return nil, err
	}
	lines := strings.Split(strings.TrimSpace(string(content)), "\n")
	entries := make(map[string]etcPasswdEntry)
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if len(line) == 0 || strings.HasPrefix(line, "#") {
			continue
		}
		entry, err := parsePasswdLine(line)
		if err != nil {
			continue
		}
		entries[entry.username] = entry
	}
	return entries, nil
}

func parsePasswdLine(line string) (etcPasswdEntry, error) {
	result := etcPasswdEntry{}
	parts := strings.Split(strings.TrimSpace(line), ":")
	if len(parts) != 7 {
		return result, fmt.Errorf("passwd line had wrong amount of parts")
	}
	result.username = strings.TrimSpace(parts[0])
	result.password = strings.TrimSpace(parts[1])

	uid, err := strconv.Atoi(parts[2])
	if err != nil {
		return result, fmt.Errorf("passwd line had badly formatted uid %s", parts[2])
	}
	result.uid = uid

	gid, err := strconv.Atoi(parts[3])
	if err != nil {
		return result, fmt.Errorf("passwd line had badly formatted gid %s", parts[2])
	}
	result.gid = gid

	result.info = strings.TrimSpace(parts[4])
	result.homedir = strings.TrimSpace(parts[5])
	result.shell = strings.TrimSpace(parts[6])
	return result, nil
}
