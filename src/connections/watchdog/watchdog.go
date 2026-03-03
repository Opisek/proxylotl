package watchdog

import (
	"bytes"
	"errors"
	"fmt"
	"mginx/connections/upstream"
	"mginx/constants"
	"mginx/models"
	"mginx/protocol/parsing"
	"mginx/protocol/payloads"
	"mginx/protocol/serializing"
	"mginx/util"
	"net"
	"time"
)

func handleUpstreamStatusConnection(conn net.Conn, res chan util.Pair[[]byte, int]) {
	defer conn.Close()

	conn.SetReadDeadline(time.Now().Add(1 * time.Second))

	var buffer bytes.Buffer

	data := make([]byte, 1024)
	for {
		n, err := conn.Read(data)
		if err != nil {
			res <- util.Pair[[]byte, int]{First: nil, Second: -1}
			return
		}

		buffer.Write(data[:n])

		packet, err := parsing.ParseHeader(buffer.Bytes())
		if err != nil {
			continue
		}
		if packet.Length > packet.ActualLength {
			continue
		}

		statusResponsePayload := packet.Payload[:len(packet.Payload)-int(packet.ActualLength-packet.Length)]

		statusResponse, err := parsing.ParseStatusResponse(statusResponsePayload)
		if err != nil {
			res <- util.Pair[[]byte, int]{First: nil, Second: -1}
			return
		}

		res <- util.Pair[[]byte, int]{First: statusResponsePayload, Second: statusResponse.Players.Online}
		return
	}
}

func checkStatus(server *models.UpstreamServer) (int, error) {
	res := make(chan util.Pair[[]byte, int])

	conn, address, err := upstream.StartClient(server.To.Hostname, server.To.Port, func(conn net.Conn) {
		handleUpstreamStatusConnection(conn, res)
	})

	if err != nil {
		return -1, errors.Join(errors.New("could not check server status"), err)
	}

	conn.Write(serializing.SerializeHandshake(payloads.Handshake{
		Version: constants.ProtocolVersion,
		Address: address,
		Port:    server.To.Port,
		Intent:  0x01,
	}))

	conn.Write(serializing.SerializeStatusRequest(payloads.StatusRequest{}))

	result := <-res
	if result.First != nil {
		server.Watchdog.LastStatusResponse = result.First
	}

	return result.Second, nil
}

func WatchUpstream(server *models.UpstreamServer) {
	lastOnlinePlayerTimestamp := time.Now()
	startupRequestTimestamp := time.Now()
	shutdownRequestTimestamp := time.Now()

	startupChannel := make(chan bool)
	server.Watchdog.RegisterWatchdog(startupChannel)

	for {
		var waitDuration time.Duration
		if server.IsTransient() {
			waitDuration = 1
		} else {
			waitDuration = 15
		}

		select {
		case <-time.After(waitDuration * time.Second):
		case <-startupChannel:
			fmt.Println("NEED TO START THE SERVER UP") // TODO: format pretty
			server.SetStarting()
			startupRequestTimestamp = time.Now()
			continue
		}

		currentTime := time.Now()
		players, err := checkStatus(server)

		if err != nil || players == -1 {
			if server.IsStartingUp() && currentTime.Sub(startupRequestTimestamp) > 60*time.Second {
				fmt.Println("UNABLE TO START THE SERVER DOWN WITHIN 60 SECONDS")
				// TODO: act accordingly
				continue
			}
			if server.IsUp() {
				fmt.Println("SERVER SHUT DOWN UNEXPECTEDLY") // TODO: format pretty
			}
			if !server.IsStartingUp() {
				server.SetDown()
			}
			continue
		} else if !server.IsUp() && !server.IsShuttingDown() {
			server.SetUp()
			if !server.IsUnknown() {
				fmt.Println("SERVER STARTED SUCCESSFULLY") // TODO: format pretty
			}
			lastOnlinePlayerTimestamp = time.Now()
			continue
		} else if server.IsShuttingDown() && currentTime.Sub(shutdownRequestTimestamp) > 60*time.Second {
			fmt.Println("UNABLE TO SHUT THE SERVER DOWN WITHIN 60 SECONDS")
			// TODO: act accordingly
			continue
		}

		if !server.IsUp() {
			continue
		}

		if players != 0 {
			lastOnlinePlayerTimestamp = time.Now()
		} else {
			if currentTime.Sub(lastOnlinePlayerTimestamp) > time.Duration(server.Watchdog.GraceTime)*time.Second {
				if server.ClientsConnecting() == 0 {
					fmt.Println("NEED TO SHUT THE SERVER DOWN") // TODO: format pretty
					server.SetStopping()
					shutdownRequestTimestamp = time.Now()
				}
			}
		}

		fmt.Printf("%s has %v online players\n", server.InternalName, players)
	}
}
