package config

import "soldr/pkg/storage"

type Config struct {
	StaticProvider *StaticProvider `json:"static_provider"`
}

type StaticProvider struct {
	Reader   storage.IFileReader
	CertsDir string
}
