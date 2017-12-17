package common

import (
	"net"
)

func AddrTo16(ip net.IP) [16]byte {
	ip = ip.To16()
	if ip == nil {
		panic("not an ip address")
	}
	var result [16]byte
	for i, x := range ip.To16() {
		result[i] = x
	}
	return result
}
