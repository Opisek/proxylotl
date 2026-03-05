package serializing

import (
	"bytes"
	"encoding/json"
	util "proxelot/protocol/internal"
	"proxelot/protocol/payloads"
)

func SerializeLoginStart(payload payloads.LoginStart) []byte {
	var buffer bytes.Buffer

	buffer.Write(util.SerializeString(payload.Name))
	buffer.Write(util.SerializeUuid(payload.Uuid))

	return SerializePacketWithHeader(0x00, buffer.Bytes())
}

func SerializeLoginSuccess(payload payloads.LoginSuccess) []byte {
	var buffer bytes.Buffer

	buffer.Write(util.SerializeUuid(payload.Uuid))
	buffer.Write(util.SerializeString(payload.Name))
	buffer.Write(util.SerializeVarInt(0))

	return SerializePacketWithHeader(0x02, buffer.Bytes())
}

func SerializeLoginDisconnect(payload payloads.LoginDisconnect) []byte {
	var buffer bytes.Buffer

	marshalledReason, _ := json.Marshal(struct {
		Reason string `json:"reason"`
	}{
		Reason: payload.Reason,
	})

	buffer.Write(marshalledReason)

	return SerializePacketWithHeader(0x00, buffer.Bytes())
}
