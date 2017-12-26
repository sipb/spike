package main

import "C"

import (
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/sipb/spike/common"
	"github.com/sipb/spike/health"
	"github.com/sipb/spike/maglev"
	"github.com/sipb/spike/tracking"
)

const (
	healthCheckNone = iota
	healthCheckHTTP
)

type serviceInfo struct {
	ip   []byte
	quit chan<- struct{}
}

type globals struct {
	services     map[string]*serviceInfo
	servicesLock sync.RWMutex
	tracker      *tracking.Cache
	maglev       *maglev.Table
}

var g globals

// Init initializes the spike health checker, connection tracker, and
// consistent hashing module.
//
//export Init
func Init() {
	g.services = make(map[string]*serviceInfo)
	g.maglev = maglev.New(maglev.SmallM)
	g.tracker = tracking.New(g.maglev.Lookup, 15*time.Minute)
}

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
	info := &serviceInfo{newIP, quit}

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
	g.servicesLock.Lock()
	defer g.servicesLock.Unlock()
	g.services[newService] = info
}

var healthCheckMap = map[string]int {
	"none": healthCheckNone,
	"http": healthCheckHTTP,
}

func AddBackendsFromConfig(file string) common.Config {
	cfg := common.ReadConfig(file)
	for _, bCfg := range cfg.Backends {
		healthCheckType, ok := healthCheckMap[bCfg.HealthCheck]
		if !ok {
			panic("Unrecognized health check type in config " + bCfg.HealthCheck)
		}
		AddBackend(bCfg.Address, bCfg.Ip, healthCheckType)
	}
	return cfg
}

// The common.Config return value can't be exported so unfortunately we need a
// separate function.

//export AddBackendsFromConfigVoid
func AddBackendsFromConfigVoid(file string) {
	AddBackendsFromConfig(file)
}

// Since Go is garbage-collected and we want to export this function and have
// Spike (Lua code, effectively C FFI for our concerns) call it to get config
// args, we have to explicitly convert our return values to C strings
// explicitly that aren't GC'd (Go does a runtime check and panics if we try to
// return Go pointers or similar, including Go strings). It's not pretty.

// Also I don't think Go FFI can export Go structs yet, so we're just returning
// five unnamed values at once...

//export AddBackendsAndGetSpikeConfig
func AddBackendsAndGetSpikeConfig(file string) (*C.char, *C.char, *C.char, *C.char, *C.char) {
	cfg := AddBackendsFromConfig(file)
	return C.CString(cfg.SrcMac), C.CString(cfg.DstMac), C.CString(cfg.Ipv4Address), C.CString(cfg.Incap), C.CString(cfg.Outcap)
}

// RemoveBackend removes a backend from the health checker.
//
//export RemoveBackend
func RemoveBackend(service string) {
	g.servicesLock.Lock()
	defer g.servicesLock.Unlock()
	info, ok := g.services[service]
	if !ok {
		return
	}
	close(info.quit)
	delete(g.services, service)
}

// Lookup determines the backend associated with a five-tuple.  It
// stores its result in output, and returns the number of bytes in the
// output.
//
//export Lookup
func Lookup(fiveTuple []byte, output []byte) int {
	backend, ok := g.tracker.Lookup(common.NewFiveTuple(fiveTuple).Hash())
	if ok {
		return copy(output, backend.IP)
	}
	return 0
}

func main() {
	fmt.Println("This main package is supposed to be compiled as a C " +
		"shared library, not as an executable.")
	os.Exit(1)
}
