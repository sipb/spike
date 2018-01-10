package config

// Read configuration stuff from a yaml file.

import (
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"log"
)

type Backend struct {
	Address     string
	IP          []byte
	HealthCheck string
}

type T struct {
	Backends    []Backend
	SrcMac      string
	DstMac      string
	IPv4Address string
	Incap       string
	Outcap      string
}

func Read(file string) T {
	var config T
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
