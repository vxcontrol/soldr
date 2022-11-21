package printer

import (
	"fmt"

	"soldr/scripts/sbh_generator/printer/file/json"
	"soldr/scripts/sbh_generator/printer/stdout"
)

type Config struct {
	Stdout *stdout.Config
	JSON   *json.Config
}

type Printer interface {
	Print(version string, tok string) error
}

func NewPrinter(c *Config) (Printer, error) {
	if c == nil {
		return nil, fmt.Errorf("passed configuration object is nil")
	}
	if c.Stdout != nil {
		p, err := stdout.NewPrinter(c.Stdout)
		if err != nil {
			return nil, fmt.Errorf("failed to initialize an stdout printer: %w", err)
		}
		return p, nil
	}
	if c.JSON != nil {
		p, err := json.NewPrinter(c.JSON)
		if err != nil {
			return nil, fmt.Errorf("failed to initialize a JSON printer: %w", err)
		}
		return p, nil
	}
	return nil, fmt.Errorf("no appropriate configuration was found")
}
