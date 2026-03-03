package phases

import (
	"errors"
	"fmt"
	"mginx/config"
	"mginx/models"
	"mginx/protocol/parsing"
	"mginx/protocol/payloads"
	"mginx/protocol/serializing"
)

func HandleStatusPhase(client *models.DownstreamClient, packet payloads.GenericPacket, conf *config.Configuration) error {
	switch packet.Id {
	case 0x00:
		err := handleStatusRequest(client, packet)
		if err != nil {
			return errors.Join(errors.New("could not parse status request packet"), err)
		}
	case 0x01:
		err := handlePingRequest(client, packet)
		if err != nil {
			return errors.Join(errors.New("could not parse ping request packet"), err)
		}
	default:
		return fmt.Errorf("invalid packet id: %v", packet.Id)
	}
	return nil
}

func handleStatusRequest(client *models.DownstreamClient, packet payloads.GenericPacket) error {
	_, err := parsing.ParseStatusRequest(packet.Payload)

	if err != nil {
		return err
	}

	client.Connection.Write(serializing.SerializeStatusResponse(client.Upstream.Watchdog.LastStatusResponse))

	return nil
}

func handlePingRequest(client *models.DownstreamClient, packet payloads.GenericPacket) error {
	payload, err := parsing.ParsePingRequest(packet.Payload)

	if err != nil {
		return err
	}

	client.Connection.Write(serializing.SerializePongResponse(payloads.PongResponse{
		Timestamp: payload.Timestamp,
	}))

	return nil
}
