package main

import "C"

import (
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/sipb/spike/common"
	"github.com/sipb/spike/config"
	"github.com/sipb/spike/health"
	"github.com/sipb/spike/maglev"
	"github.com/sipb/spike/tracking"
)

const (
	healthCheckNone = iota
	healthCheckHTTP
)

type backendInfo struct {
	ip   []byte
	quit chan<- struct{}
}

type globals struct {
	backends     map[string]*backendInfo
	backendsLock sync.RWMutex
	tracker      *tracking.Cache
	maglev       *maglev.Table
}

var g globals

// Init initializes the spike health checker, connection tracker, and
// consistent hashing module.
//
//export Init
func Init() {
	g.backends = make(map[string]*backendInfo)
	g.maglev = maglev.New(maglev.SmallM)
	g.tracker = tracking.New(g.maglev.Lookup5, 15*time.Minute)
}

// FIXME probably want to just remove this and have the interface be
// "set from a config"

// AddBackend adds a new backend to the health checker.
//
//export AddBackend
func AddBackend(service string, ip []byte, healthCheckType int) {
	// make copies of passed-in data to avoid lua gc
	newServiceBytes := make([]byte, len(service))
	copy(newServiceBytes, []byte(service))
	newService := string(newServiceBytes)
	newIP := make([]byte, len(ip))
	copy(newIP, ip)

	backends := make(chan *common.Backend, 1)
	quit := make(chan struct{})
	info := &backendInfo{newIP, quit}

	var healthCheckFunc func() bool
	switch healthCheckType {
	case healthCheckNone:
		healthCheckFunc = func() bool {
			return true
		}
	case healthCheckHTTP:
		healthCheckFunc = func() bool {
			return health.HTTP(newService, 2*time.Second)
		}
	default:
		panic("Unrecognized health check type")
	}

	health.CheckFun(healthCheckFunc,
		func() {
			down := make(chan struct{})
			backend := &common.Backend{
				IP:        newIP,
				Unhealthy: down,
			}
			backends <- backend
			g.maglev.Add(backend)
		},
		func() {
			backend := <-backends
			close(backend.Unhealthy)
			g.maglev.Remove(backend)
		},
		time.Second, 5*time.Second, quit)
	g.backendsLock.Lock()
	defer g.backendsLock.Unlock()
	g.backends[newService] = info
}

var healthCheckMap = map[string]int {
	"none": healthCheckNone,
	"http": healthCheckHTTP,
}

// Since Go is garbage-collected and we want to export this function and
// have Spike (Lua code, effectively C FFI for our concerns) call it to
// get config args, we have to explicitly convert our return values to C
// strings explicitly that aren't GC'd (Go does a runtime check and
// panics if we try to return Go pointers or similar, including Go
// strings).
//
// Also, Go FFI cannot export Go structs, so we're just returning
// several unnamed values at once.
//
//export AddBackendsAndGetSpikeConfig
func LoadConfig(file string) (*C.char, *C.char, *C.char) {
	cfg, err := config.Read(file)
	if err != nil {
		panic(err)
	}

	for _, bCfg := range cfg.Backends {
		healthCheckType, ok := healthCheckMap[bCfg.HealthCheck]
		if !ok {
			panic("Unrecognized health check type in config " + bCfg.HealthCheck)
		}
		AddBackend(bCfg.Address, bCfg.IP, healthCheckType)
	}

	return C.CString(cfg.SrcMAC), C.CString(cfg.DstMAC), C.CString(cfg.SrcIP)
}

// RemoveBackend removes a backend from the health checker.
//
//export RemoveBackend
func RemoveBackend(service string) {
	g.backendsLock.Lock()
	defer g.backendsLock.Unlock()
	info, ok := g.backends[service]
	if !ok {
		return
	}
	close(info.quit)
	delete(g.backends, service)
}

// Lookup determines the backend associated with a five-tuple.  It
// stores its result in output, and returns the number of bytes in the
// output, or -1 if no backend was found.
//
//export Lookup
func Lookup(
	src_ip, dst_ip []byte,
	src_port, dst_port uint16,
	protocol_num uint16,
	output []byte) int {
	t := common.NewFiveTuple(
		src_ip, dst_ip, src_port, dst_port, protocol_num)

	backend, ok := g.tracker.Lookup(t)
	if ok {
		return copy(output, backend.IP)
	}
	return -1
}

func main() {
	fmt.Println("This main package is supposed to be compiled as a C " +
		"shared library, not as an executable.")
	os.Exit(1)
}
