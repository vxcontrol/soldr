package config

import "soldr/internal/storage"

type Config struct {
	StaticProvider *StaticProvider `json:"static_provider"`
}

type StaticProvider struct {
	Reader   storage.IFileReader
	CertsDir string
}
