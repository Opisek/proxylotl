package payloads

type StatusRequest struct{}

type StatusResponse struct {
	Version struct {
		Name     string `json:"name"`
		Protocol int    `json:"protocol"`
	} `json:"version"`
	Players struct {
		Max    int `json:"max"`
		Online int `json:"online"`
	} `json:"players"`
	Description any    `json:"description"`
	Favicon     string `json:"favicon"`
}

type PingRequest struct {
	Timestamp uint64
}

type PongResponse struct {
	Timestamp uint64
}
