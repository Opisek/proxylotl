package phases

import (
	"errors"
	"fmt"
	"proxylotl/config"
	"proxylotl/models"
	"proxylotl/protocol/parsing"
	"proxylotl/protocol/payloads"
	"proxylotl/protocol/serializing"
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

	// Since the watchdog already pings the server periodically,
	// we can reuse the status response that we had already received.
	// This also has the side-effect of making the server appear online,
	// even when it is actually not.
	// TODO: What do we send if we haven't buffered anything yet?
	client.Connection.Write(serializing.SerializeStatusResponse(client.Upstream.Watchdog.LastStatusResponse))

	return nil
}

func handlePingRequest(client *models.DownstreamClient, packet payloads.GenericPacket) error {
	payload, err := parsing.ParsePingRequest(packet.Payload)

	if err != nil {
		return err
	}

	// Answer to pings as per the protocol.
	// Note that this might cause a somewhat incorrect round-trip time calculation on the client,
	// if the proxy and the upstream have different latencies.
	client.Connection.Write(serializing.SerializePongResponse(payloads.PongResponse{
		Timestamp: payload.Timestamp,
	}))

	return nil
}
