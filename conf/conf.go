package conf

import (
	"encoding/json"
	"fmt"
)

type Servers []*Server

type Server struct {
	Name          string `json:"name"`
	IPAddress     string `json:"ipAddress"`
	CheckInterval int    `json:"checkInterval"`
	Timeout       int    `json:"timeout"`
}

func (s *Server) String() string {
	return fmt.Sprintf("%s", s.IPAddress)
}

type Settings struct {
	Monitor *MonitorSettings
}

type MonitorSettings struct {
	CheckInterval             int `json:"checkInterval"`
	Timeout                   int `json:"timeout"`
	MaxConnections            int `json:"maxConnections"`
	ExponentialBackoffSeconds int `json:"exponentialBackoffSeconds"`
}

type Config struct {
	Servers  Servers   `json:"servers"`
	Settings *Settings `json:"settings"`
}

// NewConfig returns pointer to Config which is created from provided JSON data.
// Guarantees to be validated.
func NewConfig(jsonData []byte) *Config {
	config := &Config{}
	err := json.Unmarshal(jsonData, config)
	if err != nil {
		panic("error parsing json configuration data")
	}
	if err := ValidateAll(config); err != nil {
		panic(err)
	}
	return config
}

func (c *Config) Validate() error {
	if err := c.Settings.Validate(); err != nil {
		return fmt.Errorf("invalid settings: %v", err)
	}
	if err := c.Servers.Validate(); err != nil {
		return fmt.Errorf("invalid servers: %v", err)
	}
	return nil
}

func (s *Settings) Validate() error {
	if err := s.Monitor.Validate(); err != nil {
		return fmt.Errorf("invalid monitor settings: %v", err)
	}
	return nil
}

func (ms *MonitorSettings) Validate() error {
	// ExponentialBackoffSeconds can be 0, which means when calculated,
	// delay for notifications will always be 1 second
	if ms.CheckInterval <= 0 || ms.MaxConnections <= 0 || ms.Timeout <= 0 || ms.ExponentialBackoffSeconds < 0 {
		return fmt.Errorf("monitor settings missing")
	}
	return nil
}

func (servers Servers) Validate() error {
	if len(servers) == 0 {
		return fmt.Errorf("no servers found in config")
	}

	for _, server := range servers {
		if err := server.Validate(); err != nil {
			return fmt.Errorf("invalid server settings: %s", err)
		}

	}
	return nil
}

func (s *Server) Validate() error {
	errServerProperty := func(property string) error {
		return fmt.Errorf("missing server property %s", property)
	}
	switch {
	case s.Name == "":
		return errServerProperty("name")
	case s.IPAddress == "":
		return errServerProperty("ipAddress")
	}
	return nil
}
