package main

import "C"

import (
	"fmt"
	"github.com/sipb/spike/health"
	"github.com/sipb/spike/maglev"
	"os"
	"sync"
	"time"
)

type serviceInfo struct {
	ip   []byte
	quit chan<- struct{}
}

var globalMaglev *maglev.Table
var globalServices map[string]*serviceInfo
var globalServicesLock sync.Mutex

// Init initializes the spike health checker and consistent hashing modules.
//export Init
func Init() {
	globalMaglev = maglev.New(maglev.SmallM)
	globalServices = make(map[string]*serviceInfo)
}

// AddBackend adds a new backend to the health checker.
//export AddBackend
func AddBackend(service string, ip []byte) {
	// FIXME make copies of passed-in data to avoid lua gc
	info := &serviceInfo{ip,
		health.CheckFun(service,
			func() { globalMaglev.Add(ip) },
			func() { globalMaglev.Remove(ip) },
			100*time.Millisecond, 500*time.Millisecond)}
	globalServicesLock.Lock()
	defer globalServicesLock.Unlock()
	globalServices[service] = info
}

// RemoveBackend removes a backend from the health checker.
//export RemoveBackend
func RemoveBackend(service string) {
	globalServicesLock.Lock()
	defer globalServicesLock.Unlock()
	info, ok := globalServices[service]
	if !ok {
		return
	}
	close(info.quit)
	delete(globalServices, service)
}

// Lookup determines the backend associated with a five-tuple
//export Lookup
func Lookup(fiveTuple []byte) ([]byte, bool) {
	return globalMaglev.Lookup(fiveTuple)
}

func main() {
	fmt.Println("This main package is supposed to be compiled as a C " +
		"shared library, not as an executable.")
	os.Exit(1)
}
