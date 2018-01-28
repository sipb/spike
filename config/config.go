package config

// Read configuration from a yaml file.

import (
	"gopkg.in/yaml.v2"
	"io/ioutil"
)

type HealthCheck struct {
	// Valid types are "", "http"
	Type string

	// HTTP only
	HTTPAddr string `yaml:"http address"`
}

type Backend struct {
	Name        string
	IP          string
	HealthCheck HealthCheck
}

type Pool struct {
	VIP        string
	MaglevSize uint64 `yaml:"maglev size"`
	Backends   []Backend
}

type Interface struct {
	// Valid types are "pcap"
	Type string

	// PCAP only
	PCAPFile string `yaml:"pcap file"`
}

type T struct {
	SrcMAC string `yaml:"src mac"`
	DstMAC string `yaml:"dst mac"`
	SrcIP  string `yaml:"src ip"`

	Input  Interface
	Output Interface

	Pools []Pool
}

func Read(file string) (*T, error) {
	var config T
	dat, err := ioutil.ReadFile(file)
	if err != nil {
		return nil, err
	}
	if err = yaml.Unmarshal(dat, &config); err != nil {
		return nil, err
	}
	return &config, nil
}
