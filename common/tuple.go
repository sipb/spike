package common

import (
	"github.com/dchest/siphash"
)

const lookupKey = uint64(0xdd5d635024f19f34)

// A FiveTuple consists of source and destination IP and port, along
// with the IP protocol version number.
type FiveTuple struct {
	data []byte // TODO split this into fields
}

// NewFiveTuple constructs a new five-tuple.
func NewFiveTuple(data []byte) *FiveTuple {
	return &FiveTuple{data: data}
}

// Hash returns the consistent five-tuple hash.
func (p *FiveTuple) Hash() uint64 {
	return siphash.Hash(lookupKey, 0, p.data)
}
