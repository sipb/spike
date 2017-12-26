package common

// Read configuration stuff from a yaml file.

import (
	"log"
	"io/ioutil"
	"gopkg.in/yaml.v2"
)

type BackendConfig struct {
	Address string
	Ip []byte
	HealthCheck string
}

type Config struct {
	Backends []BackendConfig
	SrcMac string
	DstMac string
	Ipv4Address string
	Incap string
	Outcap string
}

func ReadConfig(file string) Config {
	var config Config
	dat, err := ioutil.ReadFile(file)
	if err != nil {
		log.Fatal("Cannot read config file: %v", file)
	}
	err = yaml.Unmarshal(dat, &config)
	if err != nil {
		log.Fatal("Cannot unmarshal config yaml: %v", err)
	}
	return config
}
