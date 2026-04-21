package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	ListenAddr string `json:"listen_addr"`
	ListenPort int    `yaml:"listen_port"`

	Domain         string `yaml:"domain"`
	R53ZoneID      string `yaml:"r53_zone_id"`
	AwsRegion      string `yaml:"aws_region"`
	AwsAccessKeyID string `yaml:"aws_access_key_id"`
	AwsSecretKey   string `yaml:"aws_secret_key"`

	Clients []Clients `yaml:"clients"`
}

type Clients struct {
	PSK        string `yaml:"preshared_key"`
	Hostname   string `yaml:"hostname"`
	UpdateIPv4 bool   `yaml:"update_ipv4"`
}

// LoadConfig reads the named file and unmarshals its contents into a Config.
// It first attempts JSON decoding, then falls back to YAML if JSON fails.
func LoadConfig(filename string) (*Config, error) {
	if filename == "" {
		return nil, fmt.Errorf("filename is empty")
	}

	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	cfg := &Config{
		ListenAddr: "0.0.0.0",
		ListenPort: 8080,
		AwsRegion:  "us-west-2",
	}

	// Fallback to YAML
	if err := yaml.Unmarshal(data, cfg); err == nil {
		return cfg, nil
	}

	return nil, fmt.Errorf("failed to unmarshal config as JSON or YAML")
}
