package maglev

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTableSize(t *testing.T) {
	New(1e9 + 7) // a prime
	New(1e9 + 9) // its twin
	New(SmallM)
	New(BigM)
	assert.Panics(t, func() { New(1 << 60) }, "2^60 is not prime but table created")
	assert.Panics(t, func() { New(57) }, "57 is not prime but table created")
}
