package conf

import (
	"log"

	"github.com/burntsushi/toml"
)

type ConfigVars struct {
	FacebookID     string
	FacebookSecret string
	BaseURL        string
	PGDATABASE     string
	PGUSER         string
}

var Config ConfigVars

func init() {
	if _, err := toml.DecodeFile("conf.toml", &Config); err != nil {
		log.Fatalf("Error decoding conf: %s", err)
	}
}
