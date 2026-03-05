package serializing

import (
	"bytes"
	util "proxelot/protocol/internal"
	"proxelot/protocol/payloads"
)

func SerializeKeepAlive(payload payloads.KeepAlive) []byte {
	var buffer bytes.Buffer

	buffer.Write(util.SerializeUnsignedLong(payload.Id))

	return SerializePacketWithHeader(0x04, buffer.Bytes())
}

func SerializeTransfer(payload payloads.Transfer) []byte {
	var buffer bytes.Buffer

	buffer.Write(util.SerializeString(payload.Host))
	buffer.Write(util.SerializeVarInt(uint64(payload.Port)))

	return SerializePacketWithHeader(0x0B, buffer.Bytes())
}

func SerializeShowDialog(payload payloads.ShowDialog) []byte {
	var buffer bytes.Buffer

	buffer.Write(payload.Dialog.SerializeDialog())

	return SerializePacketWithHeader(0x12, buffer.Bytes())
}
