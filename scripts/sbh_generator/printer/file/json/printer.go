package json

import (
	"encoding/json"
	"fmt"
	"os"

	"soldr/pkg/app/server/mmodule/hardening/v1/sbher"
)

type Config struct {
	File  string
	Force bool
}

type Printer struct {
	file  string
	force bool
}

func NewPrinter(c *Config) (*Printer, error) {
	if c == nil {
		return nil, fmt.Errorf("passed configuration object is nil")
	}
	if len(c.File) == 0 {
		return nil, fmt.Errorf("passed file path is empty")
	}
	return &Printer{
		file:  c.File,
		force: c.Force,
	}, nil
}

func (p *Printer) Print(version string, tok string) error {
	rawMsg, err := readSBHFileContents(p.file)
	if err != nil {
		return err
	}
	v1SectionData, err := getV1SectionFileData(rawMsg)
	if err != nil {
		return err
	}
	if err := writeNewSBH(v1SectionData, version, tok, p.force); err != nil {
		return err
	}
	if err := storeV1SectionFileData(rawMsg, v1SectionData); err != nil {
		return err
	}
	if err := replaceSBHFileContents(rawMsg, p.file); err != nil {
		return fmt.Errorf("failed to replace the SBH file contents: %w", err)
	}
	return nil
}

func readSBHFileContents(file string) (map[string]json.RawMessage, error) {
	contents, err := os.ReadFile(file)
	if err != nil {
		return nil, fmt.Errorf("failed to read the file %s: %w", file, err)
	}
	var dst map[string]json.RawMessage
	if err := json.Unmarshal(contents, &dst); err != nil {
		return nil, fmt.Errorf(
			"failed to unmarshal the contents of the file %s into a raw message structure: %w",
			file,
			err,
		)
	}
	return dst, nil
}

const v1Section = "v1"

func getV1SectionFileData(contents map[string]json.RawMessage) (sbher.SBHFileData, error) {
	var fileData sbher.SBHFileData
	v1Contents, ok := contents[v1Section]
	if !ok {
		return nil, fmt.Errorf("section %s was not found in the passed data", v1Section)
	}
	if err := json.Unmarshal(v1Contents, &fileData); err != nil {
		return nil, fmt.Errorf("failed to unmarshal the contents of the section %s: %w", v1Section, err)
	}
	return fileData, nil
}

func writeNewSBH(fileData sbher.SBHFileData, version string, token string, force bool) error {
	if _, ok := fileData[version]; ok && !force {
		return fmt.Errorf("SBH file already contains an entry for SBH of the version %s", version)
	}
	fileData[version] = []byte(token)
	return nil
}

func storeV1SectionFileData(rawMsg map[string]json.RawMessage, v1SectionData sbher.SBHFileData) error {
	newV1Section, err := json.Marshal(v1SectionData)
	if err != nil {
		return fmt.Errorf("failed to marshal the new V1 section object: %w", err)
	}
	rawMsg[v1Section] = newV1Section
	return nil
}

func replaceSBHFileContents(newContents map[string]json.RawMessage, file string) error {
	contents, err := json.MarshalIndent(newContents, "", "    ")
	if err != nil {
		return fmt.Errorf("failed to marshal the new contents object: %w", err)
	}
	if err := os.WriteFile(file, contents, 0); err != nil {
		return fmt.Errorf("failed to write the new contents to the file %s: %w", file, err)
	}
	return nil
}
