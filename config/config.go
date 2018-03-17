package config

// Read configuration from a yaml file.

import (
	"fmt"
	"gopkg.in/yaml.v2"
	"io/ioutil"
)

type HealthCheck struct {
	// valid types are "mock", "http"
	Type string
	// whether it's healthy to begin with
	Healthy bool

	// only for type "http"
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
	if config.Input.Type != "pcap" {
		panic(fmt.Sprintf("bad input type %#v", config.Input.Type))
	}
	if config.Output.Type != "pcap" {
		panic(fmt.Sprintf("bad output type %#v", config.Output.Type))
	}
	for _, p := range config.Pools {
		for _, b := range p.Backends {
			if b.HealthCheck.Type != "mock" && b.HealthCheck.Type != "http" {
				panic(fmt.Sprintf("backend %#v: bad health check type %#v",
					b.Name, b.HealthCheck.Type))
			}
		}
	}
	return &config, nil
}
