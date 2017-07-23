// Package maglev implements maglev consistent hashing.
//
// http://research.google.com/pubs/pub44824.html
package maglev

import (
	"github.com/dchest/siphash"
)

// Prime numbers of varying scale
const (
	SmallM = 65537
	BigM   = 655373
)

// Table represents a Maglev hashing table.
type Table struct {
	n           uint64 // number of backends
	m           uint64 // size of the lookup table
	permutation [][]uint64
	lookup      []int64
	backends    []string
}

// New returns a new Maglev table with the specified backends and size.
func New(backends []string, m uint64) *Table {
	// TODO clone backends
	mag := &Table{
		n:        uint64(len(backends)),
		m:        m,
		backends: backends,
	}
	mag.generatePermutations()
	mag.populate()
	return mag
}

// Add a backend to the table.
// FIXME make this idempotent
func (t *Table) Add(backend string) {
	t.backends = append(t.backends, backend)
	t.n++
	t.generatePermutations()
	t.populate()
}

// Remove a backend from the table.
func (t *Table) Remove(backend string) {
	for i, v := range t.backends {
		if v == backend {
			t.backends = append(t.backends[:i], t.backends[i+1:]...)
			t.n--
			t.generatePermutations()
			t.populate()
			return
		}
	}
}

// Lookup looks up an entry in the table and returns the associated backend.
func (t *Table) Lookup(obj string) string {
	key := siphash.Hash(0xdeadbabe, 0, []byte(obj))
	return t.backends[t.lookup[key%t.m]]
}

// TODO don't actually generate permutations; just store the linear function
func (t *Table) generatePermutations() {
	if len(t.backends) == 0 {
		return
	}

	for _, backend := range t.backends {
		b := []byte(backend)
		// FIXME use two different hashes for clarity instead of bit-fiddling
		h := siphash.Hash(0xdeadbeefcafebabe, 0, b)
		offset, skip := (h>>32)%t.m, ((h&0xffffffff)%(t.m-1) + 1)
		p := make([]uint64, t.m)
		idx := offset
		for j := uint64(0); j < t.m; j++ {
			p[j] = idx
			idx += skip
			if idx >= t.m {
				idx -= t.m
			}
		}

		t.permutation = append(t.permutation, p)
	}
}

func (t *Table) populate() {
	next := make([]uint64, t.n)
	entry := make([]int64, t.m)
	for j := range entry {
		entry[j] = -1
	}

	var n uint64
	for {
		for i := uint64(0); i < t.n; i++ {
			c := t.permutation[i][next[i]]
			for entry[c] >= 0 {
				next[i]++
				c = t.permutation[i][next[i]]
			}
			entry[c] = int64(i)
			next[i]++
			n++
			if n == t.m {
				t.lookup = entry
				return
			}
		}
	}
}
