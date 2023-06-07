package main

import (
	"encoding/json"

	"github.com/h2oai/goconfig"
	_ "github.com/h2oai/goconfig/env"
)

type Config struct {
	Host     string `cfg:"db_host" env:"DB_HOST" cfgDefault:"default.host"`
	Port     int    `cfg:"db_port" env:"DB_PORT" cfgDefault:"10101"`
	Enabled  bool   `cfg:"db_enabled" env:"DB_ENABLED"`
	ReadOnly bool   `cfg:"db_readonly" env:"DB_READONLY"`
	Options  string `cfg:"db_options" env:"DB_OPTIONS"`
}

func main() {
	config := Config{}

	goconfig.File = ".env"
	err := goconfig.Parse(&config)
	if err != nil {
		println(err)
		return
	}

	// just print struct on screen
	j, _ := json.MarshalIndent(config, "", "  ")
	println(string(j))
}
