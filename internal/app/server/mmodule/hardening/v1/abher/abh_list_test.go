package abher

import (
	"fmt"
	"testing"
)

func TestExtractABIFromDBBinaryPath(t *testing.T) {

	// Normal cases
	normalVersions := []string{
		"v1.1.0.131-cf42be4",
		"1.1.0.131-cf42be4",
		"1.1.0-cf42be4",
		"v1.1.0-cf42be4",
		"1.1.0",
		"v1.1.0",
		"1.1",
		"v1.1",
	}
	osNames := []string{
		"linux",
		"darwin",
		"windows",
	}
	architectures := []string{
		"386",
		"amd64",
	}
	filePaths := []string{
		"vxagent/%s/vxagent",
		`//世界/世界世界/世界%s世界/世界世//界/`,
		"%s",
		"vxagent/%s",
		"%s/vxagent",
	}
	notAgentSuffix := []string{
		"browser",
		"external",
	}
	getTestFn := func(filePath string, abi string) func(t *testing.T) {
		return func(t *testing.T) {
			filePath := fmt.Sprintf(filePath, abi)
			t.Log("file path: ", filePath)
			actualABI, err := ExtractABIFromDBBinaryPath(filePath)
			if err != nil {
				t.Errorf("an unexpected error occurred: %v", err)
				return
			}
			if actualABI != abi {
				t.Errorf("expected ABI %s, got: %s", abi, actualABI)
				return
			}
		}
	}
	for _, ver := range normalVersions {
		for _, osName := range osNames {
			for _, arch := range architectures {
				for _, fp := range filePaths {
					fp := fp
					abi := fmt.Sprintf("%s/%s/%s", ver, osName, arch)
					t.Run(fmt.Sprintf("normal (%s)", abi), getTestFn(fp, abi))
				}
			}
		}
		for _, s := range notAgentSuffix {
			for _, fp := range filePaths {
				fp := fp
				abi := fmt.Sprintf("%s/%s", ver, s)
				t.Run(fmt.Sprintf("normal (%s)", abi), getTestFn(fp, abi))
			}
		}
	}
	// Error cases
	const errMsg = "the file path contains an unexpected number of agentBinaryIDs (%d): %v"
	cases := []struct {
		Name       string
		InFilePath string
		OutABI     string
		OutErr     string
	}{
		{
			Name:       "err: multiple ABIs",
			InFilePath: "1.1.0.131-cf42be4/linux/amd64/v42.42/darwin/386",
			OutErr:     fmt.Sprintf(errMsg, 2, []string{"1.1.0.131-cf42be4/linux/amd64", "v1/darwin/386"}),
		},
		{
			Name:       "err: no ABIs",
			InFilePath: "bla/blablabla/somepath/someotherpass/",
			OutErr:     fmt.Sprintf(errMsg, 0, []string{}),
		},
	}

	for i, tc := range cases {
		i, tc := i, tc
		t.Run(fmt.Sprintf("test #%d: %s", i, tc.Name), func(t *testing.T) {
			actualABI, err := ExtractABIFromDBBinaryPath(tc.InFilePath)
			if err == nil {
				t.Errorf("expected an error with message \"%s\", got nil", tc.OutErr)
				return
			}
			if len(actualABI) != 0 {
				t.Errorf("expected the actual ABI to be empty, got: %s", actualABI)
				return
			}
		})
	}
}
