package downstream

import (
	"bytes"
	"errors"
	"fmt"
	"mginx/config"
	"mginx/models"
	"mginx/protocol"
	"mginx/protocol/parsing"
	"mginx/protocol/payloads"
	"mginx/util"
	"net"
	"os"
	"time"
)

func StartServer(address string, port uint16, packetQueue chan util.Pair[*models.DownstreamClient, payloads.GenericPacket], conf *config.Configuration) error {
	listener, err := net.Listen("tcp", fmt.Sprintf("%v:%v", address, port))
	if err != nil {
		return errors.Join(fmt.Errorf("could not start listening on %v:%v", address, port), err)
	}
	defer listener.Close()

	fmt.Printf("mginx is listening on %v:%v\n", address, port)

	for {
		conn, err := listener.Accept()
		if err != nil {
			continue
		}
		go handleClientConnection(conn, packetQueue)
	}
}

func handleClientConnection(conn net.Conn, packetQueue chan util.Pair[*models.DownstreamClient, payloads.GenericPacket]) {
	defer conn.Close()

	client := &models.DownstreamClient{
		Connection: conn,
	}

	var buffer bytes.Buffer

	data := make([]byte, 1024)
	for {
		conn.SetReadDeadline(time.Now().Add(10 * time.Second))
		n, err := conn.Read(data)
		if !client.IsAlive() {
			return
		}
		if err != nil {
			client.Kill()
			return
		}
		if client.IsProxying() {
			client.UpstreamConnection.Write(data[:n]) // If we are actively proxying the connection, simply pass all the data directly to the upstream
			continue
		}

		// All the following code applies only prior to the client connecting to the upstream

		buffer.Write(data[:n]) // We must buffer incoming TCP packets until we have at least one complete Minecraft packet

		// Check if we have buffered enough data to parse one or more complete packets
		for {
			packet, err := parsing.ParseHeader(buffer.Bytes())
			if err != nil {
				break
			}
			if packet.Length > packet.ActualLength {
				break
			}

			remainingPayload := packet.Payload
			cutIndex := uint64(len(remainingPayload)) - (packet.ActualLength - packet.Length)

			packet.Payload = make([]byte, cutIndex)
			copy(packet.Payload, remainingPayload[:cutIndex])

			buffer.Reset()
			buffer.Write(remainingPayload[cutIndex:])

			// Pass on a complete packet payload for furher processing
			packetQueue <- util.Pair[*models.DownstreamClient, payloads.GenericPacket]{
				First:  client,
				Second: packet,
			}
		}
	}
}

func HandlePackets(packetQueue chan util.Pair[*models.DownstreamClient, payloads.GenericPacket], conf *config.Configuration) {
	for {
		received := <-packetQueue
		client := received.First
		packet := received.Second

		if !client.IsAlive() || client.IsProxying() {
			continue
		}

		err := protocol.HandlePacket(client, packet, conf)
		if err != nil {
			fmt.Fprintln(os.Stderr, errors.Join(errors.New("could not handle client packet"), err))

			client.Kill()
		}
	}
}
