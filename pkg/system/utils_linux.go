//go:build linux
// +build linux

package system

import (
	"bufio"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"sync"

	"github.com/digitalocean/go-smbios/smbios"
)

type Feature int

const (
	_ Feature = iota

	SystemManufacturer // System manufacturer. Requires access to SMBIOS data via DMI (i.e. root privileges)
	SystemProductName  // System product name. Requires access to SMBIOS data via DMI (i.e. root privileges)
	SystemUUID         // System UUID. Makes sense for virtual machines. Requires access to SMBIOS data via DMI (i.e. root privileges)
)

const (
	EtcOsRelease    string = "/etc/os-release"
	UsrLibOsRelease string = "/usr/lib/os-release"
)

var (
	readOSInfoOnce   sync.Once
	readSMBIOSOnce   sync.Once
	osInfoReadingErr error
	smbiosReadingErr error
	osInfoAttrValues = make(map[string]string)
	smbiosAttrValues = make(map[Feature]string)

	// See SMBIOS specification https://www.dmtf.org/sites/default/files/standards/documents/DSP0134_3.3.0.pdf
	smbiosAttrTable = [...]smbiosAttribute{
		// System 0x01
		{0x01, 0x08, 0x04, SystemManufacturer, nil},
		{0x01, 0x08, 0x05, SystemProductName, nil},
		{0x01, 0x14, 0x08, SystemUUID, formatUUID},
	}
)

func getOSInfoAttr(opt string) (string, error) {
	readOSInfoOnce.Do(func() {
		osInfoReadingErr = readOSInfoAttributes(EtcOsRelease)
		if osInfoReadingErr != nil {
			osInfoReadingErr = readOSInfoAttributes(UsrLibOsRelease)
		}
	})
	if osInfoReadingErr != nil {
		return "", osInfoReadingErr
	}
	return osInfoAttrValues[opt], nil
}

func readOSInfoAttributes(filename string) error {
	lines, err := osInfoParseFile(filename)
	if err != nil {
		return err
	}

	for _, v := range lines {
		key, value, err := osInfoParseLine(v)
		if err == nil {
			osInfoAttrValues[key] = value
		}
	}
	return nil
}

func osInfoParseFile(filename string) (lines []string, err error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	return lines, scanner.Err()
}

func osInfoParseLine(line string) (key string, value string, err error) {
	err = nil

	// skip empty lines
	if len(line) == 0 {
		err = errors.New("Skipping: zero-length")
		return
	}

	// skip comments
	if line[0] == '#' {
		err = errors.New("Skipping: comment")
		return
	}

	// try to split string at the first '='
	splitString := strings.SplitN(line, "=", 2)
	if len(splitString) != 2 {
		err = errors.New("Can not extract key=value")
		return
	}

	// trim white space from key and value
	key = splitString[0]
	key = strings.Trim(key, " ")
	value = splitString[1]
	value = strings.Trim(value, " ")

	// Handle double quotes
	if strings.ContainsAny(value, `"`) {
		first := string(value[0:1])
		last := string(value[len(value)-1:])

		if first == last && strings.ContainsAny(first, `"'`) {
			value = strings.TrimPrefix(value, `'`)
			value = strings.TrimPrefix(value, `"`)
			value = strings.TrimSuffix(value, `'`)
			value = strings.TrimSuffix(value, `"`)
		}
	}

	// expand anything else that could be escaped
	value = strings.Replace(value, `\"`, `"`, -1)
	value = strings.Replace(value, `\$`, `$`, -1)
	value = strings.Replace(value, `\\`, `\`, -1)
	value = strings.Replace(value, "\\`", "`", -1)
	return
}

func formatUUID(b []byte) (string, error) {
	return fmt.Sprintf("%0x-%0x-%0x-%0x-%0x", b[:4], b[4:6], b[6:8], b[8:10], b[10:16]), nil
}

func getSMBIOSAttr(feat Feature) (string, error) {
	readSMBIOSOnce.Do(func() {
		smbiosReadingErr = readSMBIOSAttributes()
	})
	if smbiosReadingErr != nil {
		return "", smbiosReadingErr
	}
	return smbiosAttrValues[feat], nil
}

func readSMBIOSAttributes() error {
	smbiosAttrIndex := buildSMBIOSAttrIndex()
	rc, _, err := smbios.Stream()
	if err != nil {
		return fmt.Errorf("unable to open SMBIOS info: %v", err)
	}
	defer rc.Close()
	structures, err := smbios.NewDecoder(rc).Decode()
	if err != nil {
		return fmt.Errorf("unable to decode SMBIOS info: %v", err)
	}
	for _, s := range structures {
		attrList := smbiosAttrIndex[int(s.Header.Type)]
		for _, attr := range attrList {
			val, err := attr.readValueString(s)
			if err != nil {
				return fmt.Errorf("unable to read SMBIOS attribute '%v' of structure type 0x%0x: %v",
					attr.feature, s.Header.Type, err)
			}
			smbiosAttrValues[attr.feature] = val
		}
	}
	return nil
}

func buildSMBIOSAttrIndex() map[int][]*smbiosAttribute {
	res := make(map[int][]*smbiosAttribute)
	for i := range smbiosAttrTable {
		attr := &smbiosAttrTable[i]
		res[attr.structType] = append(res[attr.structType], attr)
	}
	return res
}

type smbiosAttribute struct {
	structType      int
	structMinLength int
	offset          int

	feature Feature

	format func(data []byte) (string, error)
}

func (attr *smbiosAttribute) readValueString(s *smbios.Structure) (string, error) {
	if len(s.Formatted) < attr.structMinLength {
		return "", nil
	}
	if attr.format != nil {
		const headerSize = 4
		return attr.format(s.Formatted[attr.offset-headerSize:])
	}
	return attr.getString(s)
}

func (attr *smbiosAttribute) getString(s *smbios.Structure) (string, error) {
	const headerSize = 4
	strNo := int(s.Formatted[attr.offset-headerSize])
	strNo -= 1
	if strNo < 0 || strNo >= len(s.Strings) {
		return "", fmt.Errorf("invalid string no")
	}
	return s.Strings[strNo], nil
}

func getOSName() string {
	if value, err := getOSInfoAttr("NAME"); err != nil {
		return ""
	} else {
		return value
	}
}

func getOSVer() string {
	if value, err := getOSInfoAttr("VERSION_ID"); err != nil {
		return "0"
	} else {
		return value
	}
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
	machineID := strings.TrimSpace(strings.Trim(string(id), "\n"))

	// root privileges are required to access attributes, the process will be skipped in case of insufficient privileges
	smbiosAttrs := [...]Feature{SystemUUID, SystemManufacturer, SystemProductName}
	for _, attr := range smbiosAttrs {
		attrVal, err := getSMBIOSAttr(attr)
		if err != nil || strings.TrimSpace(attrVal) == "" {
			continue
		}
		machineID = fmt.Sprintf("%s:%s", machineID, strings.ToLower(attrVal))
	}

	return machineID, nil
}
