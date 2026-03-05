package parsing

import (
	"errors"
	"fmt"
	util "proxelot/protocol/internal"
	"proxelot/protocol/payloads"
)

func ParseKeepAlive(buffer []byte) (payloads.KeepAlive, error) {
	id, buffer, err := util.ParseUnsignedLong(buffer)
	if err != nil {
		return payloads.KeepAlive{},
			errors.Join(errors.New("could not parse keepalive id"), err)
	}

	if len(buffer) != 0 {
		return payloads.KeepAlive{},
			fmt.Errorf("remaining bytes at the end of payload: %v", len(buffer))
	}

	return payloads.KeepAlive{
		Id: id,
	}, nil
}
