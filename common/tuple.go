package common

import (
	"bytes"
	"encoding/binary"

	"github.com/dchest/siphash"
)

const lookupKey = uint64(0xdd5d635024f19f34)

// A FiveTuple consists of source and destination IP and port, along
// with the IP protocol version number.
type FiveTuple struct {
	Src_ip, Dst_ip     [16]byte
	Src_port, Dst_port uint16
	Protocol_num       uint16
}

// NewFiveTuple constructs a new five-tuple.
func NewFiveTuple(
	src_ip, dst_ip []byte,
	src_port, dst_port uint16,
	protocol_num uint16) FiveTuple {
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

	return FiveTuple{
		Src_ip:       AddrTo16(src_ip),
		Dst_ip:       AddrTo16(dst_ip),
		Src_port:     src_port,
		Dst_port:     dst_port,
		Protocol_num: protocol_num,
	}
}

func (t *FiveTuple) encode() []byte {
	b := new(bytes.Buffer)
	binary.Write(b, binary.LittleEndian, t.Src_ip)
	binary.Write(b, binary.LittleEndian, t.Dst_ip)
	binary.Write(b, binary.LittleEndian, t.Src_port)
	binary.Write(b, binary.LittleEndian, t.Dst_port)
	binary.Write(b, binary.LittleEndian, t.Protocol_num)
	return b.Bytes()
}

// Hash returns the five-tuple hash.
func (t *FiveTuple) Hash() uint64 {
	return siphash.Hash(lookupKey, 0, t.encode())
}
