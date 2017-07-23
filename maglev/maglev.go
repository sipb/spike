// Package maglev implements maglev consistent hashing.
//
// http://research.google.com/pubs/pub44824.html
package maglev

import (
	"github.com/dchest/siphash"
	"sync"
)

// Prime numbers of varying scale
const (
	SmallM = 65537
	BigM   = 655373
)

const (
	offsetKey = uint64(0x35d53c5371bdf886)
	skipKey   = uint64(0x9e1dbc702649df3a)
	lookupKey = uint64(0xdd5d635024f19f34)
)

type permutation struct {
	offset uint64
	skip   uint64
}

// Table represents a Maglev hashing table.
type Table struct {
	m            uint64 // size of the lookup table
	backends     []string
	weights      []uint
	permutations []permutation
	lookup       []int
	mutex sync.RWMutex
}

// New returns a new Maglev table with the specified size.
func New(m uint64) *Table {
	mag := &Table{
		m:            m,
	}
	return mag
}

// SetWeight sets the weight of the given backend to weight, adding it
// to the table if necessary.
func (t *Table) SetWeight(backend string, weight uint) {
	t.mutex.Lock()
	defer t.mutex.Unlock()

	for i, b := range t.backends {
		if backend == b {
			t.weights[i] = weight
			t.populate()
			return
		}
	}

	t.backends = append(t.backends, backend)
	t.weights = append(t.weights, weight)
	offset := siphash.Hash(offsetKey, 0, []byte(backend)) % t.m
	skip := siphash.Hash(skipKey, 0, []byte(backend))%(t.m-1) + 1
	t.permutations = append(t.permutations, permutation{offset, skip})

	t.populate()
}

// Compact deletes entries from the table with weight 0
func (t *Table) Compact() {
	t.mutex.Lock()
	defer t.mutex.Unlock()

	var newBackends []string
	var newWeights []uint
	var newPermutations []permutation

	for i, w := range t.weights {
		if w > 0 {
			newBackends = append(newBackends, t.backends[i])
			newWeights = append(newWeights, w)
			newPermutations = append(newPermutations, t.permutations[i])
		}
	}
	t.backends = newBackends
	t.weights = newWeights
	t.permutations = newPermutations
	t.populate()
}

// Add adds a backend to the table with weight 1.
func (t *Table) Add(backend string) {
	t.SetWeight(backend, 1)
}

// Remove removes a backend from the table by setting its weight to 0.
func (t *Table) Remove(backend string) {
	t.SetWeight(backend, 0)
}

// Lookup looks up an entry in the table and returns the associated
// backend.
func (t *Table) Lookup(obj string) (string, bool) {
	t.mutex.RLock()
	defer t.mutex.RUnlock()

	if t.lookup == nil {
		return "", false
	}
	key := siphash.Hash(lookupKey, 0, []byte(obj))
	return t.backends[t.lookup[key%t.m]], true
}

func (t *Table) populate() {
	nonzero := false
	for _, w := range t.weights {
		if w > 0 {
			nonzero = true
			break
		}
	}
	if !nonzero {
		t.lookup = nil
		return
	}

	loc := make([]uint64, len(t.backends))
	for i, p := range t.permutations {
		loc[i] = p.offset
	}
	entry := make([]int, t.m)
	for j := range entry {
		entry[j] = -1
	}
	var inserted uint64

	for {
		for i, w := range t.weights {
			for j := uint(0); j < w; j++ {
				c := loc[i]
				for entry[c] >= 0 {
					c += t.permutations[i].skip
					if c >= t.m {
						c -= t.m
					}
				}
				entry[c] = i
				c += t.permutations[i].skip
				if c >= t.m {
					c -= t.m
				}
				loc[i] = c

				inserted++
				if inserted == t.m {
					t.lookup = entry
					return
				}
			}
		}
	}
}
