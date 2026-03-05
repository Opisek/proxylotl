package serializing

import (
	"bytes"
	util "proxylotl/protocol/internal"
)

func SerializePacketWithHeader(id uint64, payload []byte) []byte {
	var buffer bytes.Buffer

	buffer.Write(util.SerializeVarInt(id))
	buffer.Write(payload)

	serializedLen := util.SerializeVarInt(uint64(buffer.Len()))

	return append(serializedLen, buffer.Bytes()...)
}
