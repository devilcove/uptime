package uptime

import (
	"encoding/json"
	"log"
	"os"
	"path/filepath"
)

type Config struct {
	DBFile string
}

var config *Config

func GetConfig() *Config {
	if config != nil {
		return config
	}
	xdg, ok := os.LookupEnv("XDG_CONFIG_HOME")
	if !ok {
		home, _ := os.UserHomeDir()
		xdg = filepath.Join(home, ".config/uptime")
	}

	file, err := os.ReadFile(filepath.Join(xdg, "config"))
	if err != nil {
		log.Println("unable to read config file", err)
		return nil
	}
	if err := json.Unmarshal(file, &config); err != nil {
		log.Println("json", err)
		return nil
	}
	return config

}
