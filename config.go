package main

import "github.com/BurntSushi/toml"

type Config struct {
	Devices []struct {
		Attr []struct {
			IP   string `toml:"ip"`
			Name string `toml:"name"`
		} `toml:"attr"`
	} `toml:"devices"`
	Nfilters []struct {
		Attr []struct {
			Ips  []string `toml:"ips"`
			Name string   `toml:"name"`
		} `toml:"attr"`
	} `toml:"nfilters"`
}

func ReadConfig(filename string) (*Config, error) {
	var cfg Config
	if _, err := toml.DecodeFile(filename, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}
