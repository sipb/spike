// Package maglev implements maglev consistent hashing.
//
// http://research.google.com/pubs/pub44824.html
package maglev

import (
	"math/big"
	"sort"
	"sync"

	"github.com/dchest/siphash"

	"github.com/sipb/spike/common"
)

// Prime numbers of varying scale
const (
	SmallM = 65537
	BigM   = 655373
)

const (
	offsetKey = uint64(0x35d53c5371bdf886)
	skipKey   = uint64(0x9e1dbc702649df3a)
)

type permutation struct {
	weight uint
	offset uint64
	skip   uint64
}

// Table represents a Maglev hashing table.
type Table struct {
	m            uint64 // size of the lookup table
	permutations map[*common.Backend]permutation
	lookup       []*common.Backend
	mutex        sync.RWMutex
}

// New returns a new Maglev table with the specified size.
func New(m uint64) *Table {
	if !(&big.Int{}).SetUint64(m).ProbablyPrime(0) {
		panic("m is not prime")
	}
	return &Table{
		m: m,
		permutations: make(map[*common.Backend]permutation),
	}
}

// A Config is a mapping from backends to weights.
type Config map[*common.Backend]uint

// Reconfig reconfigures the table with the given backend weight
// configuration.
func (t *Table) Reconfig(c Config) {
	t.mutex.Lock()
	defer t.mutex.Unlock()

	t.permutations = make(map[*common.Backend]permutation)
	for b, w := range c {
		if b == nil {
			panic("nil backend in config")
		}
		if w == 0 {
			continue
		}
		t.permutations[b] = permutation{
			weight: w,
			offset: siphash.Hash(offsetKey, 0, []byte(b.IP)) % t.m,
			skip:   siphash.Hash(skipKey, 0, []byte(b.IP))%(t.m-1) + 1,
		}
	}
	t.populate()
}

// SetWeight sets the weight of the given backend to weight, adding it
// to the table if necessary.
func (t *Table) SetWeight(backend *common.Backend, weight uint) {
	t.mutex.Lock()
	defer t.mutex.Unlock()

	if backend == nil {
		panic("backend is nil")
	}

	if weight == 0 {
		delete(t.permutations, backend)
		t.populate()
		return
	}

	p, ok := t.permutations[backend]
	if ok {
		p.weight = weight
		t.permutations[backend] = p
	} else {
		t.permutations[backend] = permutation{
			weight: weight,
			offset: siphash.Hash(offsetKey, 0, []byte(backend.IP)) % t.m,
			skip:   siphash.Hash(skipKey, 0, []byte(backend.IP))%(t.m-1) + 1,
		}
	}
	t.populate()
}

// Add adds a backend to the table with weight 1.
func (t *Table) Add(backend *common.Backend) {
	t.SetWeight(backend, 1)
}

// Remove removes a backend from the table.
func (t *Table) Remove(backend *common.Backend) {
	t.mutex.Lock()
	defer t.mutex.Unlock()
	delete(t.permutations, backend)
	t.populate()
}

// Lookup looks up a key in the table and returns the associated
// backend, or false if there are no backends.
func (t *Table) Lookup(key uint64) (*common.Backend, bool) {
	t.mutex.RLock()
	defer t.mutex.RUnlock()

	if t.lookup == nil {
		return nil, false
	}
	return t.lookup[key%t.m], true
}

func (t *Table) populate() {
	nonzero := false
	for _, p := range t.permutations {
		if p.weight > 0 {
			nonzero = true
			break
		}
	}
	if !nonzero {
		t.lookup = nil
		return
	}

	type bstate struct {
		backend *common.Backend
		loc     uint64
		permutation
	}

	state := make([]bstate, 0, len(t.permutations))
	for b, p := range t.permutations {
		state = append(state, bstate{b, p.offset, p})
	}
	// sort state to guarantee consistency given identical configurations
	sort.Slice(state, func(i, j int) bool {
		return state[i].offset < state[j].offset
	})

	entry := make([]*common.Backend, t.m)
	for j := range entry {
		entry[j] = nil
	}

	var inserted uint64
	for {
		for i, s := range state {
			for j := uint(0); j < s.weight; j++ {
				c := s.loc
				for entry[c] != nil {
					c += s.skip
					if c >= t.m {
						c -= t.m
					}
				}
				entry[c] = s.backend
				c += s.skip
				if c >= t.m {
					c -= t.m
				}
				state[i].loc = c

				inserted++
				if inserted == t.m {
					t.lookup = entry
					return
				}
			}
		}
	}
}
