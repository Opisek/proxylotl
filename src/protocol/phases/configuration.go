package phases

import (
	"errors"
	"fmt"
	"proxylotl/config"
	"proxylotl/models"
	util "proxylotl/protocol/internal"
	"proxylotl/protocol/parsing"
	"proxylotl/protocol/payloads"
	"proxylotl/protocol/serializing"
	"time"
)

func HandleConfigurationPhase(client *models.DownstreamClient, packet payloads.GenericPacket, conf *config.Configuration) error {
	switch packet.Id {
	case 0x04:
		err := handleClientKeepAlive(client, packet)
		if err != nil {
			return errors.Join(errors.New("could not parse keepalive packet"), err)
		}
	default:
		// We don't handle configuration packets from the clients, since we just want them to wait for the server to start up
		return nil
	}
	return nil
}

func handleClientKeepAlive(client *models.DownstreamClient, packet payloads.GenericPacket) error {
	payload, err := parsing.ParseKeepAlive(packet.Payload)

	if err != nil {
		return err
	}

	if payload.Id != client.ExpectedKeepalive {
		return fmt.Errorf("keepalive id %v received but expected %v", payload.Id, client.ExpectedKeepalive)
	}

	client.ExpectedKeepalive, err = util.GetRandomLong()
	if err != nil {
		return errors.Join(errors.New("could not generate a random keepalive ID"), err)
	}

	// Periodically send keepalives to clients that are waiting for the upstream server to start
	go func() {
		time.Sleep(5 * time.Second)

		if !client.IsAlive() || client.IsProxying() {
			return
		}

		client.Connection.Write(serializing.SerializeKeepAlive(payloads.KeepAlive{
			Id: client.ExpectedKeepalive,
		}))
	}()

	return nil
}
