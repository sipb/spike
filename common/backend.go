package common

// Backend keeps track of a backend's IP address and whether the backend
// has become unhealthy.
type Backend struct {
	IP []byte

	// Unhealthy is closed when the backend is determined to be unhealthy.
	Unhealthy chan struct{}
}
