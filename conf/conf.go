package conf

import (
	"log"
	"os"
	"strings"

	"github.com/burntsushi/toml"
)

type ConfigVars struct {
	FacebookID         string
	FacebookSecret     string
	BaseURL            string
	PostgresDataSource string
}

var Config ConfigVars

func init() {
	if _, err := toml.DecodeFile("conf.toml", &Config); err != nil {
		log.Fatalf("Error decoding conf: %s", err)
	}
	Config.PostgresDataSource = strings.Replace(Config.PostgresDataSource,
		"$POSTGRES_PORT_5432_TCP_ADDR", os.Getenv("POSTGRES_PORT_5432_TCP_ADDR"), -1)
}
