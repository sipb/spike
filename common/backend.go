package common

type Backend struct {
	IP          []byte
	Unhealthy   chan struct{}
}
