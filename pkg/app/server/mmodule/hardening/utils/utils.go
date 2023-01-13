package utils

import (
	"fmt"
	"regexp"
)

var agentIDRegexp = regexp.MustCompile("^[0-9a-f]{32}$")

func IsAgentIDValid(id string) error {
	if !agentIDRegexp.MatchString(id) {
		return fmt.Errorf("passed agent ID is not valid")
	}
	return nil
}
