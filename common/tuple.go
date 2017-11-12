package common

import (
	"bytes"
	"encoding/binary"
	"net"

	"github.com/dchest/siphash"
)

const lookupKey = uint64(0xdd5d635024f19f34)

// A FiveTuple consists of source and destination IP and port, along
// with the IP protocol version number.
type FiveTuple struct {
	src_ip, dst_ip     [16]byte
	src_port, dst_port uint16
	protocol_num       uint16
}

// NewFiveTuple constructs a new five-tuple.
func NewFiveTuple(
	src_ip, dst_ip []byte,
	src_port, dst_port uint16,
	protocol_num uint16) FiveTuple {
	var src_ip_a, dst_ip_a [16]byte
	switch protocol_num {
	case L3_IPV4:
		if len(src_ip) != 4 || len(dst_ip) != 4 {
			panic("IPv4 address must be length 4")
		}
	case L3_IPV6:
		if len(src_ip) != 16 || len(dst_ip) != 16 {
			panic("IPv6 address must be length 16")
		}
	default:
		panic("invalid protocol number")
	}
	for i, x := range net.IP(src_ip).To16() {
		src_ip_a[i] = x
	}
	for i, x := range net.IP(dst_ip).To16() {
		dst_ip_a[i] = x
	}

	return FiveTuple{
		src_ip:       src_ip_a,
		dst_ip:       dst_ip_a,
		src_port:     src_port,
		dst_port:     dst_port,
		protocol_num: protocol_num,
	}
}

func (t *FiveTuple) encode() []byte {
	b := new(bytes.Buffer)
	binary.Write(b, binary.LittleEndian, t.src_ip)
	binary.Write(b, binary.LittleEndian, t.dst_ip)
	binary.Write(b, binary.LittleEndian, t.src_port)
	binary.Write(b, binary.LittleEndian, t.dst_port)
	binary.Write(b, binary.LittleEndian, t.protocol_num)
	return b.Bytes()
}

// Hash returns the five-tuple hash.
func (t *FiveTuple) Hash() uint64 {
	return siphash.Hash(lookupKey, 0, t.encode())
}
