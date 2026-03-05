package serializing

import (
	"bytes"
	util "proxylotl/protocol/internal"
	"proxylotl/protocol/payloads"
)

func SerializeHandshake(payload payloads.Handshake) []byte {
	var buffer bytes.Buffer

	buffer.Write(util.SerializeVarInt(payload.Version))
	buffer.Write(util.SerializeString(payload.Address))
	buffer.Write(util.SerializeUnsignedShort(payload.Port))
	buffer.Write(util.SerializeVarInt(payload.Intent))

	return SerializePacketWithHeader(0x00, buffer.Bytes())
}
