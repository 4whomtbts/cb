package config

import "circuitbreaker_slave/types"

type CircuitBreakerConfig struct {
	NodeName string `yaml:"nodeName"`
	Type string `yaml:"type"`
	Port int `yaml:"port"`
	CircuitBreakerLevel string `yaml:"circuitBreakerLevel"`
	WatchIntervalSec       int `yaml:"watchIntervalSec"`
	HealthCheckIntervalSec int `yaml:"healthCheckIntervalSec"`
	CircuitBreakerIntervalSec int `yaml:"circuitBreakerIntervalSec"`
	MasterServer string `yaml:"masterServer"`
	Email string `yaml:"email"`
	EmailPassword string `yaml:"emailPassword"`
	EmailReceivers []string `yaml:"emailReceivers"`
	Nodes []string `yaml:"nodes"`
	Admins []*types.Admin `yaml:"admins"`
	Exporters []Exporter `yaml:"exporters"`
}

type Exporter struct {
	Name string `yaml:"name"`
	Label string `yaml:"label"`
	Url string	`yaml:"url"`
	Config map[string]string `yaml:"config"`
}