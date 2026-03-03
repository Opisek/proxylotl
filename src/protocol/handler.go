package protocol

import (
	"errors"
	"fmt"
	"mginx/config"
	"mginx/models"
	"mginx/protocol/payloads"
	"mginx/protocol/phases"
)

func HandlePacket(client *models.DownstreamClient, packet payloads.GenericPacket, conf *config.Configuration) error {
	switch client.GamePhase {
	case 0x00:
		fmt.Println("initial phase")
		err := phases.HandleHandshakePhase(client, packet, conf)
		if err != nil {
			return errors.Join(errors.New("could not handle packet in handshake phase"), err)
		}
	case 0x01:
		fmt.Println("status phase")
		err := phases.HandleStatusPhase(client, packet, conf)
		if err != nil {
			return errors.Join(errors.New("could not handle packet in status phase"), err)
		}
	case 0x02:
		fmt.Println("login phase")
		err := phases.HandleLoginPhase(client, packet, conf)
		if err != nil {
			return errors.Join(errors.New("could not handle packet in login phase"), err)
		}
	case 0x04:
		fmt.Println("configuration phase")
		err := phases.HandleConfigurationPhase(client, packet, conf)
		if err != nil {
			return errors.Join(errors.New("could not handle packet in configuration phase"), err)
		}
	default:
		return fmt.Errorf("invalid game phase: %v", client.GamePhase)
	}

	return nil
}
