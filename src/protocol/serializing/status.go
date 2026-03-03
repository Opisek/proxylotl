package serializing

import (
	"bytes"
	util "mginx/protocol/internal"
	"mginx/protocol/payloads"
)

func SerializeStatusRequest(payloads payloads.StatusRequest) []byte {
	return SerializePacketWithHeader(0x00, []byte{})
}

func SerializeStatusResponse(rawPayload []byte) []byte {
	return SerializePacketWithHeader(0x00, rawPayload)
}

func SerializePongResponse(payload payloads.PongResponse) []byte {
	var buffer bytes.Buffer

	buffer.Write(util.SerializeUnsignedLong(payload.Timestamp))

	return SerializePacketWithHeader(0x01, buffer.Bytes())
}
