package config

import (
	"os"
)

type ProjectConfig struct {
	Servers    []string      `yaml:"servers,flow"`
	InOutLinks []LinkMethods `yaml:"inputOutput,flow"`
	OutLinks   []LinkMethods `yaml:"output,flow"`
}

type LinkMethods struct {
	Name    string   `yaml:"name"`
	Key     string   `yaml:"key"`
	Servers []string // duplicate for processing in input & output
}

type ServerConfig struct {
	Ip           string `yaml:"ip"`
	Port         string `yaml:"port"`
	JbossPort    string `yaml:"jboss_port"`
	Timeout      int    `yaml:"timeout"` // timeout *10 sec
	ServerSecret Secret `yaml:"server_secret"`
	OfsSecret    Secret `yaml:"ofs_secret"`
	TafjSecret   Secret `yaml:"tafj_secret"`
	Scripts      []struct {
		Name    string `yaml:"name"`
		Type    string `yaml:"type"`
		Script  string `yaml:"script"`
		Message string `yaml:"message"`
	} `yaml:"scripts,flow"`
	Commands []struct {
		Name    string   `yaml:"name"`
		Scripts []string `yaml:"scripts,flow"`
	} `yaml:"commands,flow"`
}

type Secret struct {
	User     string `yaml:"user"`
	Password string `yaml:"password"`
}

var ConfigPath = getEnv("CONFIG_PATH", ".")

// Simple helper function to read an environment or return a default value
func getEnv(key string, defaultVal string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}

	return defaultVal
}
