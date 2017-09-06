package maglev

import (
	"fmt"
	"math/rand"
	"testing"

	"github.com/sipb/spike/common"
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

func TestBasicFunctionality(t *testing.T) {
	table := New(SmallM)

	backends := make([]common.Backend, 6)
	for i := 0; i < len(backends); i++ {
		backends[i] = common.Backend{IP: []byte{0, 0, 0, byte(i)}}
	}

	table.Add(&backends[0])
	table.Add(&backends[1])
	table.Add(&backends[2])

	table.SetWeight(&backends[3], 2)
	table.SetWeight(&backends[3], 3)

	table.Add(&backends[4])
	table.Remove(&backends[4])

	table.Add(&backends[5])
	table.SetWeight(&backends[5], 0)

	rand.Seed(42)
	freq := make(map[*common.Backend]uint)
	for i := 0; i < 1e4; i++ {
		cur, _ := table.Lookup(rand.Uint64())
		freq[cur] = freq[cur] + 1
	}

	assert.Equal(t, 4, len(freq), "There should only be three backends.")
	for i := 0; i < 4; i++ {
		assert.True(t, freq[&backends[i]] > 0, fmt.Sprintf("backends[%d] not hit", i))
	}
}
