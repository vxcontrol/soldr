package config

import (
	"soldr/pkg/filestorage"
)

type Config struct {
	StaticProvider *StaticProvider `json:"static_provider"`
}

type StaticProvider struct {
	Reader   filestorage.Reader
	CertsDir string
}
