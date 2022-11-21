package main

import (
	"fmt"
	"log"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	conf, err := NewConfig()
	if err != nil {
		return err
	}
	if conf.CertsSuite.Active {
		log.Println("generating a certificates suite")
		if err := generateCertificatesSuite(conf.CertsSuite, *conf.ExpirationTimeConfig); err != nil {
			return fmt.Errorf("failed to generate a certificates suite: %w", err)
		}
	}
	return nil
}
