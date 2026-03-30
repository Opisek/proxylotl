package models

import (
	"errors"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"sync"

	"github.com/goccy/go-yaml"
)

var serverAddressRegex = regexp.MustCompile(`^([^:]+)(:(\d+))?$`)

func (addr Address) String() string {
	if addr.Port == uint16(25565) {
		return addr.Hostname
	}
	return fmt.Sprintf("%v:%v", addr.Hostname, addr.Port)
}

func (addr Address) MarshalYAML() ([]byte, error) {
	if addr.Port == uint16(25565) {
		return []byte(addr.Hostname), nil
	}
	return fmt.Appendf(nil, "%v:%v", addr.Hostname, addr.Port), nil
}

func (addr *Address) UnmarshalYAML(b []byte) error {
	var str string
	if err := yaml.Unmarshal(b, &str); err != nil {
		return err
	}

	matches := serverAddressRegex.FindAllStringSubmatch(str, -1)

	if matches == nil || len(matches) != 1 || len(matches[0]) != 4 {
		return fmt.Errorf("malformed hostname or IP address: %v", str)
	}

	if len(matches[0][3]) == 0 {
		addr.Port = 25565
	} else {
		value, err := strconv.ParseUint(matches[0][3], 10, 16)

		if err != nil {
			return errors.Join(errors.New("could not parse port"), err)
		}

		addr.Port = uint16(value)
	}

	addr.Hostname = matches[0][1]

	return nil
}

type Address struct {
	Hostname string
	Port     uint16
}

const (
	serverStateUnmanaged = iota
	serverStateUnknown
	serverStateDown
	serverStateStarting
	serverStateUp
	serverStateStopping
)

type watchdogConfiguration struct {
	StartCommand string `yaml:"start"`
	StopCommand  string `yaml:"stop"`
	GraceTime    uint   `yaml:"grace"`

	LastStatusResponse []byte `yaml:""`
	statusPath         string `yaml:""`

	startupChannel chan bool `yaml:""`
	startupQueued  bool      `yaml:""`
}

func (watchdog *watchdogConfiguration) IsManaged() bool {
	return watchdog.GraceTime != 0
}

func (watchdog *watchdogConfiguration) RegisterWatchdog(startupChannel chan bool, statusPath string) {
	watchdog.startupChannel = startupChannel
	watchdog.statusPath = statusPath
	watchdog.LoadStatus()
}

func (watchdog *watchdogConfiguration) SaveStatus() {
	file, err := os.Create(watchdog.statusPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Could not open server status file %v for writing: %v\n", watchdog.statusPath, err)
		return
	}

	defer func() {
		if err := file.Close(); err != nil {
			fmt.Fprintf(os.Stderr, "Could not close server status file %v: %v\n", watchdog.statusPath, err)
		}
	}()

	if _, err = file.Write(watchdog.LastStatusResponse); err != nil {
		fmt.Fprintf(os.Stderr, "Could not write server status to file %v: %v\n", watchdog.statusPath, err)
	}
}

func (watchdog *watchdogConfiguration) LoadStatus() {
	file, err := os.Open(watchdog.statusPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Could not open server status file %v for reading: %v\n", watchdog.statusPath, err)
		return
	}

	defer func() {
		if err := file.Close(); err != nil {
			fmt.Fprintf(os.Stderr, "Could not close server status file %v: %v\n", watchdog.statusPath, err)
		}
	}()

	if _, err = file.Read(watchdog.LastStatusResponse); err != nil {
		fmt.Fprintf(os.Stderr, "Could not read server status from file %v: %v\n", watchdog.statusPath, err)
	}
}

type UpstreamServer struct {
	InternalName string                `yaml:""`
	From         []Address             `yaml:"from"`
	To           Address               `yaml:"to"`
	Redirect     bool                  `yaml:"redirect"`
	Watchdog     watchdogConfiguration `yaml:"watchdog"`

	serverState            int                          `yaml:""`
	serverStartupCallbacks map[*DownstreamClient]func() `yaml:""`
	connectedClientsCount  int                          `yaml:""`
	serverStateLock        sync.RWMutex                 `yaml:""`
	serverDownChannel      chan bool                    `yaml:""`
}

func (server *UpstreamServer) IsUnknown() bool {
	server.serverStateLock.RLock()
	defer server.serverStateLock.RUnlock()

	return server.serverState == serverStateUnknown
}

func (server *UpstreamServer) SetUnknown() {
	server.serverStateLock.Lock()
	defer server.serverStateLock.Unlock()

	server.serverState = serverStateUnknown
	server.serverStartupCallbacks = make(map[*DownstreamClient]func())
}

func (server *UpstreamServer) IsUp() bool {
	server.serverStateLock.RLock()
	defer server.serverStateLock.RUnlock()

	return server.serverState == serverStateUp
}

func (server *UpstreamServer) SetUp() {
	server.serverStateLock.Lock()
	defer server.serverStateLock.Unlock()

	server.serverState = serverStateUp

	// One the server is up, make sure to notify all clients that ary trying to connect
	if len(server.serverStartupCallbacks) != 0 {
		for _, callback := range server.serverStartupCallbacks {
			go callback()
		}
		server.serverStartupCallbacks = make(map[*DownstreamClient]func())
	}
}

func (server *UpstreamServer) IsDown() bool {
	server.serverStateLock.RLock()
	defer server.serverStateLock.RUnlock()

	return server.serverState == serverStateDown
}

func (server *UpstreamServer) SetDown() {
	server.serverStateLock.Lock()
	defer server.serverStateLock.Unlock()

	server.serverState = serverStateDown

	// Once the server is down, if we are waiting to starting it back up,
	// then we may now continue with that process.
	if server.serverDownChannel != nil {
		server.serverDownChannel <- true
	}
}

func (server *UpstreamServer) IsStartingUp() bool {
	server.serverStateLock.RLock()
	defer server.serverStateLock.RUnlock()

	return server.serverState == serverStateStarting
}

func (server *UpstreamServer) SetStarting() {
	server.serverStateLock.Lock()
	defer server.serverStateLock.Unlock()

	server.serverState = serverStateStarting
}

func (server *UpstreamServer) IsShuttingDown() bool {
	server.serverStateLock.RLock()
	defer server.serverStateLock.RUnlock()

	return server.serverState == serverStateStopping
}

func (server *UpstreamServer) SetStopping() {
	server.serverStateLock.Lock()
	defer server.serverStateLock.Unlock()

	server.serverState = serverStateStopping
}

func (server *UpstreamServer) IsTransient() bool {
	server.serverStateLock.RLock()
	defer server.serverStateLock.RUnlock()

	return server.serverState == serverStateUnknown || server.serverState == serverStateStarting || server.serverState == serverStateStopping
}

func (server *UpstreamServer) Connect(client *DownstreamClient, callback func(), callbackIfClosed func()) bool {
	if !server.Watchdog.IsManaged() {
		callback()
		return true
	}

	// We remember that we counted this client as connecting a few lines further down
	client.MarkConneting()

	server.serverStateLock.Lock()
	defer server.serverStateLock.Unlock()

	// We remember that a player is trying to connect so that we don't turn the server off
	server.connectedClientsCount += 1

	// If the server is already up, let the client connect immediately
	if server.serverState == serverStateUp {
		callback()
		return true
	}

	// Otherwise, we need to use a different callback once the server starts up
	server.serverStartupCallbacks[client] = callbackIfClosed

	// Make sure that we only try to start the server once
	if !server.Watchdog.startupQueued {
		server.Watchdog.startupQueued = true
		go func() {
			server.serverStateLock.Lock()
			defer func() {
				server.Watchdog.startupQueued = false
				server.serverStateLock.Unlock()
			}()

			// If the server is currently stopping, we need to wait for this to be over
			if server.serverState == serverStateStopping {
				server.serverDownChannel = make(chan bool)
				server.serverStateLock.Unlock()
				<-server.serverDownChannel
				server.serverStateLock.Lock()

				// If the client has disconnected during the above wait, we abort the start-up
				if server.connectedClientsCount == 0 {
					return
				}
			}

			// If the server is currently down, we trigger a start-up
			if server.serverState == serverStateDown {
				server.Watchdog.startupChannel <- true
			}
		}()
	}

	return false
}

func (server *UpstreamServer) ClientDisconnected(client *DownstreamClient) {
	if !server.Watchdog.IsManaged() {
		return
	}

	marked := client.IsConnecting()

	server.serverStateLock.Lock()
	defer server.serverStateLock.Unlock()

	// If we had counted this client as willing to connect before,
	// we must decrease the counter once its connection closes.
	if marked {
		server.connectedClientsCount -= 1
	}

	// Avoid memory leaks
	delete(server.serverStartupCallbacks, client)
}

func (server *UpstreamServer) ClientsConnecting() int {
	server.serverStateLock.RLock()
	defer server.serverStateLock.RUnlock()

	return server.connectedClientsCount
}
