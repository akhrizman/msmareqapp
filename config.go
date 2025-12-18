package main

import (
	"log"
	"os"
)

type AppConfig struct {
	DefaultPasswordSuffix string
}

var Config AppConfig

func LoadConfig() {
	Config.DefaultPasswordSuffix = os.Getenv("DEFAULT_PASSWORD_SUFFIX")

	if Config.DefaultPasswordSuffix == "" {
		log.Fatal("DEFAULT_PASSWORD_SUFFIX is required")
	}
}
