package common

// Read configuration stuff from a yaml file.

import (
	"log"
	"gopkg.in/yaml.v2"
)

var data = `
backends:
    - address: http://cheesy-fries.mit.edu/health
      ip: [1,2,3,4]
    - address: http://strawberry-habanero.mit.edu/health
      ip: [5,6,7,8]
`

type BackendConfig struct {
	Address string
	Ip []byte
}

type Config struct {
	Backends []BackendConfig
}

func ReadConfig() Config {
	var config Config
	err := yaml.Unmarshal([]byte(data), &config)
	if err != nil {
		log.Fatal("Cannot unmarshal config yaml: %v", err)
	}
	for _, backend := range config.Backends {
		log.Printf("Backend!")
		log.Printf("  Address is %v", backend.Address)
		log.Printf("  IP is %v", backend.Ip)
	}
	return config
}
