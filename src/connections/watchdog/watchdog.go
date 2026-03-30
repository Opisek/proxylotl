package watchdog

import (
	"bytes"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"proxylotl/connections/upstream"
	"proxylotl/constants"
	"proxylotl/models"
	"proxylotl/protocol/parsing"
	"proxylotl/protocol/payloads"
	"proxylotl/protocol/serializing"
	"proxylotl/util"
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

		// Similar to buffering client packets
		buffer.Write(data[:n])

		// Check if we have buffered a complete paket already
		packet, err := parsing.ParseHeader(buffer.Bytes())
		if err != nil {
			continue
		}
		if packet.Length > packet.ActualLength {
			continue
		}

		// Parse the server status
		statusResponsePayload := packet.Payload[:len(packet.Payload)-int(packet.ActualLength-packet.Length)]

		statusResponse, err := parsing.ParseStatusResponse(statusResponsePayload)
		if err != nil {
			res <- util.Pair[[]byte, int]{First: nil, Second: -1}
			return
		}

		// Report the server status alongside the raw payload bytes,
		// so that we may reuse the latter when clients contact us for this upstream's status.
		res <- util.Pair[[]byte, int]{First: statusResponsePayload, Second: statusResponse.Players.Online}
		return
	}
}

func checkStatus(server *models.UpstreamServer) (int, error) {
	res := make(chan util.Pair[[]byte, int])

	// Connect to the upstream
	conn, address, err := upstream.StartClient(server.To.Hostname, server.To.Port, func(conn net.Conn) {
		handleUpstreamStatusConnection(conn, res)
	})

	if err != nil {
		return -1, errors.Join(errors.New("could not check server status"), err)
	}

	// Perform a handshake
	conn.Write(serializing.SerializeHandshake(payloads.Handshake{
		Version: constants.ProtocolVersion,
		Address: address,
		Port:    server.To.Port,
		Intent:  0x01,
	}))

	// Ask the upstream for its status
	conn.Write(serializing.SerializeStatusRequest(payloads.StatusRequest{}))

	result := <-res
	if result.First != nil && (len(server.Watchdog.LastStatusResponse) == 0 || !bytes.Equal(server.Watchdog.LastStatusResponse, result.First)) {
		server.Watchdog.LastStatusResponse = result.First // If we have received a valid response, we cache it so that we may serve our clients
		server.Watchdog.SaveStatus()
	}

	return result.Second, nil
}

func WatchUpstream(server *models.UpstreamServer) {
	lastOnlinePlayerTimestamp := time.Now()
	startupRequestTimestamp := time.Now()
	shutdownRequestTimestamp := time.Now()

	startupChannel := make(chan bool)
	server.Watchdog.RegisterWatchdog(startupChannel, filepath.Join("/app/status", fmt.Sprintf("%v.json", server.InternalName)))

	for {
		var waitDuration time.Duration
		if server.IsTransient() {
			waitDuration = 1 // If the server is currently starting up or shutting down (or if the proxy has just spun up), ping the upstream frequently
		} else {
			waitDuration = 15 // If the server is either up or down, ping it less frequently
		}

		select {
		case <-time.After(waitDuration * time.Second): // Wait for the next scheduled ping
		case <-startupChannel: // Interrupt the wait-time if we receive a start-up request
			fmt.Printf("Starting up server %v\n", server.InternalName)
			server.SetStarting()
			err := util.RunCommand(server.Watchdog.StartCommand)
			if err != nil {
				fmt.Fprintln(os.Stderr, errors.Join(fmt.Errorf("could not run the start-up command for server %v", server.InternalName), err))
			}
			startupRequestTimestamp = time.Now()
			continue
		}

		currentTime := time.Now()

		// Check the current server status
		players, err := checkStatus(server)

		if err != nil || players == -1 {
			// If the server is currently down and...

			// ...we had been trying to start the server up, but we did not succeed, try again.
			if server.IsStartingUp() && currentTime.Sub(startupRequestTimestamp) > constants.TransientServerRetryTime*time.Second {
				fmt.Printf("Could not start up server %v, retrying\n", server.InternalName)
				err := util.RunCommand(server.Watchdog.StartCommand)
				if err != nil {
					fmt.Fprintln(os.Stderr, errors.Join(fmt.Errorf("could not run the start-up command for server %v", server.InternalName), err))
				}
				startupRequestTimestamp = time.Now()
				continue
			}

			// ...the watchdog thinks that the server is up, then it must have crashed or been otherwise shut down.
			if server.IsUp() {
				fmt.Printf("Server %v has shut down unexpectedly\n", server.InternalName)
			}

			// ...we were trying to shut the server down, then we have succeeded.
			if !server.IsDown() && !server.IsStartingUp() {
				if !server.IsUnknown() {
					fmt.Printf("Server %v has shut down successfully\n", server.InternalName)
				}
				server.SetDown()
			}
			continue
		} else if !server.IsUp() && !server.IsShuttingDown() {
			// If the server is currently up and we were trying to start it, then we have succeeded
			if !server.IsUnknown() {
				fmt.Printf("Server %v has started up successfully\n", server.InternalName)
			}
			server.SetUp()
			lastOnlinePlayerTimestamp = time.Now()
			continue
		} else if server.IsShuttingDown() && currentTime.Sub(shutdownRequestTimestamp) > 60*time.Second {
			// If we were trying to shut the server down, but we did not succeed, try again
			fmt.Printf("Could not shut down server %v, retrying\n", server.InternalName)
			err := util.RunCommand(server.Watchdog.StopCommand)
			if err != nil {
				fmt.Fprintln(os.Stderr, errors.Join(fmt.Errorf("could not run the shut-down command for server %v", server.InternalName), err))
			}
			shutdownRequestTimestamp = time.Now()
			continue
		}

		if !server.IsUp() {
			continue
		}

		// The following part is only relevant for servers that are currently up

		if players != 0 {
			// Note the last timestamp we had seen an online player
			lastOnlinePlayerTimestamp = time.Now()
		} else {
			// If we had not seed an online player for the duration of the grace period and nobody is actively trying to connect, shut the server down.
			if currentTime.Sub(lastOnlinePlayerTimestamp) > time.Duration(server.Watchdog.GraceTime)*time.Second {
				if server.ClientsConnecting() == 0 {
					fmt.Printf("Shutting down server %v due to inactivity\n", server.InternalName)
					server.SetStopping()
					err := util.RunCommand(server.Watchdog.StopCommand)
					if err != nil {
						fmt.Fprintln(os.Stderr, errors.Join(fmt.Errorf("could not run the shut-down command for server %v", server.InternalName), err))
					}
					shutdownRequestTimestamp = time.Now()
				}
			}
		}
	}
}
