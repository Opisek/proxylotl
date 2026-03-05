package upstream

import (
	"errors"
	"fmt"
	"net"
	"proxelot/models"
	"strings"
	"time"
)

func StartClient(address string, port uint16, callback func(conn net.Conn)) (net.Conn, string, error) {
	if net.ParseIP(address) == nil {
		_, addrs, err := net.LookupSRV("minecraft", "tcp", address)
		if err == nil && len(addrs) != 0 {
			address = strings.TrimSuffix(addrs[0].Target, ".")
		}
	}

	conn, err := net.Dial("tcp", fmt.Sprintf("%v:%v", address, port))
	if err != nil {
		return nil, "", errors.Join(fmt.Errorf("could not connect to %v:%v", address, port), err)
	}

	go callback(conn)

	return conn, address, nil
}

func handleUpstreamProxyConnection(conn net.Conn, client *models.DownstreamClient) {
	defer conn.Close()

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

		client.Connection.Write(data[:n]) // We simply proxy all data directly to the client
	}
}

func ProxyConnection(client *models.DownstreamClient) (string, error) {
	conn, address, err := StartClient(client.Upstream.To.Hostname, client.Upstream.To.Port, func(conn net.Conn) {
		handleUpstreamProxyConnection(conn, client)
	})

	if err != nil {
		return "", errors.Join(errors.New("could not connect to the upstream"), err)
	}

	client.UpstreamConnection = conn
	client.EnableProxying()

	return address, nil
}
