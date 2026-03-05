package parsing

import (
	"encoding/json"
	"errors"
	"fmt"
	util "proxylotl/protocol/internal"
	"proxylotl/protocol/payloads"
)

func ParseStatusRequest(buffer []byte) (payloads.StatusRequest, error) {
	if len(buffer) != 0 {
		return payloads.StatusRequest{},
			fmt.Errorf("remaining bytes at the end of payload: %v", len(buffer))
	}

	return payloads.StatusRequest{}, nil
}

func ParsePingRequest(buffer []byte) (payloads.PingRequest, error) {
	timestamp, buffer, err := util.ParseUnsignedLong(buffer)
	if err != nil {
		return payloads.PingRequest{},
			errors.Join(errors.New("could not parse ping request timestamp"), err)
	}

	if len(buffer) != 0 {
		return payloads.PingRequest{},
			fmt.Errorf("remaining bytes at the end of payload: %v", len(buffer))
	}

	return payloads.PingRequest{
		Timestamp: timestamp,
	}, nil
}

func ParseStatusResponse(buffer []byte) (payloads.StatusResponse, error) {
	jsonString, buffer, err := util.ParseString(buffer)
	if err != nil {
		return payloads.StatusResponse{},
			errors.Join(errors.New("could not parse json string"), err)
	}

	if len(buffer) != 0 {
		return payloads.StatusResponse{},
			fmt.Errorf("remaining bytes at the end of payload: %v", len(buffer))
	}

	var payload payloads.StatusResponse

	err = json.Unmarshal([]byte(jsonString), &payload)
	if err != nil {
		return payloads.StatusResponse{},
			errors.Join(errors.New("could not unmarshal json object"), err)
	}

	return payload, nil
}
