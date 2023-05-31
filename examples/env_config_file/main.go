package main

import (
	"encoding/json"

	"github.com/h2oai/goconfig"
	_ "github.com/h2oai/goconfig/env"
)

type Config struct {
	Host     string   `cfg:"db_host" cfgDefault:"default.host"`
	Port     int      `cfg:"db_port" cfgDefault:"10101"`
	Enabled  bool     `cfg:"db_enabled"`
	ReadOnly bool     `cfg:"db_readonly"`
	Options  []string `cfg:"db_options"`
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
