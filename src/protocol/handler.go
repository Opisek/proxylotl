package protocol

import (
	"errors"
	"fmt"
	"proxelot/config"
	"proxelot/models"
	"proxelot/protocol/payloads"
	"proxelot/protocol/phases"
)

func HandlePacket(client *models.DownstreamClient, packet payloads.GenericPacket, conf *config.Configuration) error {
	// Depending on which game phase the packet has been sent in, pass it on to one of the different handlers
	switch client.GamePhase {
	case 0x00:
		err := phases.HandleHandshakePhase(client, packet, conf)
		if err != nil {
			return errors.Join(errors.New("could not handle packet in handshake phase"), err)
		}
	case 0x01:
		err := phases.HandleStatusPhase(client, packet, conf)
		if err != nil {
			return errors.Join(errors.New("could not handle packet in status phase"), err)
		}
	case 0x02:
		err := phases.HandleLoginPhase(client, packet, conf)
		if err != nil {
			return errors.Join(errors.New("could not handle packet in login phase"), err)
		}
	case 0x04:
		err := phases.HandleConfigurationPhase(client, packet, conf)
		if err != nil {
			return errors.Join(errors.New("could not handle packet in configuration phase"), err)
		}
	default:
		return fmt.Errorf("invalid game phase: %v", client.GamePhase)
	}

	return nil
}
