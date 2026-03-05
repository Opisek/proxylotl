package parsing

import (
	"errors"
	"fmt"
	util "proxelot/protocol/internal"
	"proxelot/protocol/payloads"
)

func ParseHandshake(buffer []byte) (payloads.Handshake, error) {
	version, buffer, err := util.ParseVarInt(buffer)
	if err != nil {
		return payloads.Handshake{},
			errors.Join(errors.New("could not parse client version"), err)
	}

	address, buffer, err := util.ParseString(buffer)
	if err != nil {
		return payloads.Handshake{},
			errors.Join(errors.New("could not parse client address"), err)
	}

	port, buffer, err := util.ParseUnsignedShort(buffer)
	if err != nil {
		return payloads.Handshake{},
			errors.Join(errors.New("could not parse client port"), err)
	}

	intent, buffer, err := util.ParseVarInt(buffer)
	if err != nil {
		return payloads.Handshake{},
			errors.Join(errors.New("could not parse client intent"), err)
	}

	if len(buffer) != 0 {
		return payloads.Handshake{},
			fmt.Errorf("remaining bytes at the end of payload: %v", len(buffer))
	}

	return payloads.Handshake{
		Version: version,
		Address: address,
		Port:    port,
		Intent:  intent,
	}, nil
}
