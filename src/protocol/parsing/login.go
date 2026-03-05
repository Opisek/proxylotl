package parsing

import (
	"errors"
	"fmt"
	util "proxylotl/protocol/internal"
	"proxylotl/protocol/payloads"
)

func ParseLoginStart(buffer []byte) (payloads.LoginStart, error) {
	name, buffer, err := util.ParseString(buffer)
	if err != nil {
		return payloads.LoginStart{},
			errors.Join(errors.New("could not parse client username"), err)
	}

	uuid, buffer, err := util.ParseUuid(buffer)
	if err != nil {
		return payloads.LoginStart{},
			errors.Join(errors.New("could not parse client uuid"), err)
	}

	if len(buffer) != 0 {
		return payloads.LoginStart{},
			fmt.Errorf("remaining bytes at the end of payload: %v", len(buffer))
	}

	return payloads.LoginStart{
		Name: name,
		Uuid: uuid,
	}, nil
}

func ParseLoginAcknowledged(buffer []byte) (payloads.LoginAcknowledged, error) {
	if len(buffer) != 0 {
		return payloads.LoginAcknowledged{},
			fmt.Errorf("remaining bytes at the end of payload: %v", len(buffer))
	}

	return payloads.LoginAcknowledged{}, nil
}
