package config

import (
	"fmt"
	"math"
	"os"
	"proxelot/models"
	"strconv"

	"github.com/goccy/go-yaml"
)

type Configuration struct {
	Servers    map[string]*models.UpstreamServer `yaml:"servers"`
	fromToServ map[string]string                 `yaml:""`
}

func (conf *Configuration) GetUpstream(hostname string, port uint16) *models.UpstreamServer {
	fromAddr := models.Address{
		Hostname: hostname,
		Port:     port,
	}

	upstreamName, ok := conf.fromToServ[fromAddr.String()]
	if !ok {
		return nil
	}

	upstream, ok := conf.Servers[upstreamName]
	if !ok {
		return nil
	}

	return upstream
}

func ReadConfig(configPath string) (*Configuration, error) {
	var conf Configuration

	data, err := os.ReadFile(configPath)

	if err != nil {
		return nil, err
	}

	err = yaml.Unmarshal(data, &conf)

	if err != nil {
		return nil, err
	}

	conf.fromToServ = make(map[string]string)

	for serverName, server := range conf.Servers {
		server.InternalName = serverName
		if server.Watchdog.IsManaged() {
			server.SetUnknown()
		}

		for _, from := range server.From {
			_, ok := conf.fromToServ[from.String()]
			if ok {
				panic(fmt.Errorf("duplicate from address: %v", from.String()))
			}
			conf.fromToServ[from.String()] = serverName
		}
	}

	return &conf, nil
}

func ParseEnvironmentalVariables() (uint16, string, error) {
	var port uint16
	portEnvVar := os.Getenv("PORT")
	if portEnvVar == "" {
		port = 25565
	} else {
		portInt, err := strconv.Atoi(portEnvVar)
		if err != nil || portInt > math.MaxUint16 {
			return 0, "", fmt.Errorf("invalid port: %v", portEnvVar)
		}
		port = uint16(portInt)
	}

	configPath := os.Getenv("CONFIG")
	if configPath == "" {
		configPath = "../config.yml"
	}

	return port, configPath, nil
}
