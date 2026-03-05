package phases

import (
	"errors"
	"fmt"
	"proxelot/config"
	"proxelot/connections/upstream"
	"proxelot/models"
	"proxelot/protocol/parsing"
	"proxelot/protocol/payloads"
	"proxelot/protocol/serializing"
)

func HandleHandshakePhase(client *models.DownstreamClient, packet payloads.GenericPacket, conf *config.Configuration) error {
	switch packet.Id {
	case 0x00:
		err := handleClientHandshake(client, packet, conf)
		if err != nil {
			return errors.Join(errors.New("could not parse handshake packet"), err)
		}
	default:
		return fmt.Errorf("invalid packet id: %v", packet.Id)
	}
	return nil
}

func handleClientHandshake(client *models.DownstreamClient, packet payloads.GenericPacket, conf *config.Configuration) error {
	payload, err := parsing.ParseHandshake(packet.Payload)

	if err != nil {
		return err
	}

	switch payload.Intent {
	case 0x01:
		client.GamePhase = 0x01 // Status
	case 0x02:
		fallthrough
	case 0x03:
		client.GamePhase = 0x02 // Connecting
	default:
		return fmt.Errorf("invalid handshake intent: %v", payload.Intent)
	}

	client.Version = payload.Version
	client.Address = payload.Address
	client.Port = payload.Port
	client.Upstream = conf.GetUpstream(client.Address, client.Port) // Figure out which upstream server the client wants to access

	// If the address in the client's payload does not match any upstream, we immediately kill the connection.
	if client.Upstream == nil {
		fmt.Printf("Client %v attempted to connect to an inexistent server %v\n", client.Connection.RemoteAddr().String(), client.Address)
		return fmt.Errorf(`no upstream found for address "%v:%v"`, client.Address, client.Port)
	}

	// If the client wants to see the server status and the server is unmanaged, we simply proxy the connection.
	// This is because transfers are not a thing for status requests.
	// If the server is managed, then the watchdog already provides us with recent status responses, so we use that instead.
	if client.GamePhase == 0x01 && !client.Upstream.Watchdog.IsManaged() {
		actualAddress, err := upstream.ProxyConnection(client)
		if err != nil {
			return errors.Join(errors.New("could not proxy connection"), err)
		}

		client.UpstreamConnection.Write(serializing.SerializeHandshake(payloads.Handshake{
			Version: client.Version,
			Address: actualAddress,
			Port:    client.Port,
			Intent:  0x01,
		}))

		client.UpstreamConnection.Write(serializing.SerializeStatusRequest(payloads.StatusRequest{}))
	}

	return nil
}
