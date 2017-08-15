package common

import (
	"github.com/dchest/siphash"
)

const lookupKey = uint64(0xdd5d635024f19f34)

type FiveTuple struct {
	data []byte // TODO split this into fields
}

func NewFiveTuple(data []byte) *FiveTuple {
	return &FiveTuple{data: data}
}

func (p *FiveTuple) Hash() uint64 {
	return siphash.Hash(lookupKey, 0, p.data)
}
