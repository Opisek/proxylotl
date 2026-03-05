package main

import (
	"errors"
	"fmt"
	"proxylotl/config"
	"proxylotl/connections/downstream"
	"proxylotl/connections/watchdog"
	"proxylotl/models"
	"proxylotl/protocol/payloads"
	"proxylotl/util"
	"time"
)

func main() {
	// Parse all configuration and environmental variables
	port, configPath, err := config.ParseEnvironmentalVariables()
	if err != nil {
		fmt.Println(errors.Join(errors.New("could not parse environmental variables"), err))
		return
	}
	conf, err := config.ReadConfig(configPath)
	if err != nil {
		fmt.Println(errors.Join(errors.New("could not parse configuration file"), err))
		return
	}

	// Watch managed servers
	for _, server := range conf.Servers {
		if server.Watchdog.IsManaged() {
			go watchdog.WatchUpstream(server)
		}
	}
	time.Sleep(3 * time.Second) // Let the watchdog initialize for every server

	// Channel for fully buffered packets to be processed further
	packetQueue := make(chan util.Pair[*models.DownstreamClient, payloads.GenericPacket])

	// Handle fully buffered packets
	go downstream.HandlePackets(packetQueue, conf)

	// Handle connections
	downstream.StartServer("", port, packetQueue, conf)
}
