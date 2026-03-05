package payloads

import (
	"proxylotl/protocol/dialog"
)

type KeepAlive struct {
	Id uint64
}

type Transfer struct {
	Host string
	Port uint16
}

type ShowDialog struct {
	Dialog dialog.Dialog
}
