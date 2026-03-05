package parsing

import (
	util "proxylotl/protocol/internal"
	"proxylotl/protocol/payloads"
)

func ParseHeader(buffer []byte) (payloads.GenericPacket, error) {
	packetLen, buffer, err := util.ParseVarInt(buffer)
	if err != nil {
		return payloads.GenericPacket{}, err
	}

	actualLen := uint64(len(buffer))

	packetId, buffer, err := util.ParseVarInt(buffer)
	if err != nil {
		return payloads.GenericPacket{}, err
	}

	return payloads.GenericPacket{
		Length:       packetLen,
		Id:           packetId,
		Payload:      buffer,
		ActualLength: actualLen,
	}, nil
}
