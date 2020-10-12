package main

import "github.com/BurntSushi/toml"


type Config struct {
	SMTP struct {
		Server string `toml:"server"`
		Port   int    `toml:"port"`
		Mail   string `toml:"mail"`
		Pass   string `toml:"pass"`
	} `toml:"smtp"`
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
