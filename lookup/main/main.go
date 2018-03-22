package main

import "C"

import (
	"fmt"
	"os"

	"github.com/sipb/spike/lookup"
	"github.com/sipb/spike/common"
	"github.com/sipb/spike/config"
)

var gLookup *lookup.T

// Init initializes the spike health checker, connection tracker, and
// consistent hashing module.
//
//export Init
func Init() {
	gLookup = lookup.New()
}

// Since Go is garbage-collected and we want to export this function and
// have Spike (Lua code, effectively C FFI for our concerns) call it to
// get config args, we have to explicitly convert our return values to C
// strings explicitly that aren't GC'd (Go does a runtime check and
// panics if we try to return Go pointers or similar, including Go
// strings).
//
// TODO: as of go 1.1.0, cgo now supports Go strings
//
// Also, Go FFI cannot export Go structs, so we're just returning
// several unnamed values at once.
//
// Returns source MAC, dest MAC, and source IP
//
//export LoadConfig
func LoadConfig(file string) (*C.char, *C.char, *C.char) {
	cfg, err := config.Read(file)
	if err != nil {
		panic(err)
	}

	gLookup.Reconfig(cfg)

	return C.CString(cfg.SrcMAC), C.CString(cfg.DstMAC), C.CString(cfg.SrcIP)
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

	backend, ok := gLookup.Lookup(t)
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
