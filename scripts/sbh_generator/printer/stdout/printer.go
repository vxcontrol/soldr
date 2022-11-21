package stdout

import (
	"encoding/base64"
	"fmt"
	"os"
)

type Config struct{}

type Printer struct{}

func NewPrinter(c *Config) (*Printer, error) {
	if c == nil {
		return nil, fmt.Errorf("passed configuration object is nil")
	}
	return &Printer{}, nil
}

func (p *Printer) Print(_ string, tok string) error {
	fmt.Fprintf(os.Stdout, "%s", base64.StdEncoding.EncodeToString([]byte(tok)))
	return nil
}
