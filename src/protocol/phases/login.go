package phases

import (
	"errors"
	"fmt"
	"proxelot/config"
	"proxelot/connections/upstream"
	"proxelot/models"
	"proxelot/protocol/dialog"
	util "proxelot/protocol/internal"
	"proxelot/protocol/parsing"
	"proxelot/protocol/payloads"
	"proxelot/protocol/serializing"
	"time"
)

func HandleLoginPhase(client *models.DownstreamClient, packet payloads.GenericPacket, conf *config.Configuration) error {
	switch packet.Id {
	case 0x00:
		err := handleClientLoginStart(client, packet)
		if err != nil {
			return errors.Join(errors.New("could not parse login start packet"), err)
		}
	case 0x03:
		err := handleClientLoginAcknowledged(client, packet)
		if err != nil {
			return errors.Join(errors.New("could not parse login acknowledged packet"), err)
		}
	default:
		return fmt.Errorf("invalid packet id: %v", packet.Id)
	}
	return nil
}

func handleClientLoginStart(client *models.DownstreamClient, packet payloads.GenericPacket) error {
	payload, err := parsing.ParseLoginStart(packet.Payload)

	if err != nil {
		return err
	}

	client.Username = payload.Name
	client.Uuid = payload.Uuid

	fmt.Printf("Client %v (%v) is connecting to server %v (%v)\n", client.Connection.RemoteAddr().String(), client.Username, client.Upstream.InternalName, client.Address)

	// If, we do not support redirects, we want to create a proxy channel if the server is up
	if !client.Upstream.Redirect {
		reconnectTriggerChannel := make(chan bool)

		alreadyUp := client.Upstream.Connect(client, func() {
			// If we are able to connect immediately, create a proxy channel
			actualAddress, err := upstream.ProxyConnection(client)
			if err != nil {
				client.Connection.Write(serializing.SerializeLoginDisconnect(payloads.LoginDisconnect{
					Reason: "Could not connect to the server",
				}))
			}

			client.UpstreamConnection.Write(serializing.SerializeHandshake(payloads.Handshake{
				Version: client.Version,
				Address: actualAddress,
				Port:    client.Port,
				Intent:  0x02,
			}))

			client.UpstreamConnection.Write(serializing.SerializeLoginStart(payload))

			client.EnableProxying()
		}, func() {
			// If we first need to go into a waiting queue, we will have to reconnect once the server is up.
			// This is because proxying must start before the server enables compression or encryption,
			// so we artificially go back to the handshake phase.
			<-reconnectTriggerChannel
			client.Connection.Write(serializing.SerializeTransfer(payloads.Transfer{
				Host: client.Address,
				Port: client.Port,
			}))
			client.StartTransfer()
		})

		if alreadyUp {
			return nil
		}

		// Since transfers can only be done in the login phase,
		// we must ensure that this client has reached the phase before we try to send a transfer packet.
		client.RegisterLoginPhaseChannel(reconnectTriggerChannel)
	}

	// If we use redirects or the server is not yet up, go into the login phase
	client.Connection.Write(serializing.SerializeLoginSuccess(payloads.LoginSuccess{
		Name: client.Username,
		Uuid: client.Uuid,
	}))

	return nil
}

func handleClientLoginAcknowledged(client *models.DownstreamClient, packet payloads.GenericPacket) error {
	_, err := parsing.ParseLoginAcknowledged(packet.Payload)

	if err != nil {
		return err
	}

	client.GamePhase = 0x04

	client.LoginFinished()
	if client.Upstream.Redirect {
		transferFunc := func() {
			client.Connection.Write(serializing.SerializeTransfer(payloads.Transfer{
				Host: client.Upstream.To.Hostname,
				Port: client.Upstream.To.Port,
			}))
			client.StartTransfer()

			// Once the server is up, the client must transfer.
			// If they don't, we eventually kill the connection.
			client.Connection.SetWriteDeadline(time.Now().Add(10 * time.Second))
		}

		alreadyConnected := client.Upstream.Connect(client, transferFunc, transferFunc)

		if alreadyConnected {
			return nil
		}
	}

	client.ExpectedKeepalive, err = util.GetRandomLong()
	if err != nil {
		return errors.Join(errors.New("could not generate a random keepalive ID"), err)
	}

	client.Connection.Write(serializing.SerializeKeepAlive(payloads.KeepAlive{
		Id: client.ExpectedKeepalive,
	}))

	client.Connection.Write(serializing.SerializeShowDialog(payloads.ShowDialog{Dialog: dialog.Dialog{
		Title:       "Server Startup",
		Description: "The server is starting up.\n\nYou will be connected shortly.\n\nPlease wait...",
	}}))

	return nil
}
