// Package Maglev implements maglev consistent hashing
/*
   http://research.google.com/pubs/pub44824.html
*/
package maglev

import (
	"github.com/dchest/siphash"
)

const (
	SmallM = 65537
	BigM   = 655373
)

type Table struct {
	n           uint64 //size of VIP backends
	m           uint64 //size of the lookup table
	permutation [][]uint64
	lookup      []int64
	names    []string
}

func New(names []string, m uint64) *Table {
	mag := &Table{
		n:			uint64(len(names)),
		m:			m,
		names:	names,
	}
	mag.generatePermutations()
	mag.populate()
	return mag
}

func (t *Table) Add(name string) {
	t.names = append(t.names, name)
	t.n = uint64(len(t.names))
	t.generatePermutations()
	t.populate()
}

func (t *Table) Remove(name string) {
	for i, v := range t.names {
		if v == name {
			t.names = append(t.names[:i], t.names[i+1:]...)
			break
		}
	}

	t.n = uint64(len(t.names))
	t.generatePermutations()
	t.populate()
}

func (t *Table) Lookup(obj string) string {
	key := siphash.Hash(0xdeadbabe, 0, []byte(obj))
	return t.names[t.lookup[key%t.m]]
}

func (t *Table) generatePermutations() {
	if len(t.names) == 0 {
		return
	}

	for _, name := range t.names {
		b := []byte(name)
		h := siphash.Hash(0xdeadbeefcafebabe, 0, b)
		offset, skip := (h>>32)%t.m, ((h&0xffffffff)%(t.m-1) + 1)
		p := make([]uint64, t.m)
		idx := offset
		for j := uint64(0); j < t.m; j++ {
			p[j] = idx
			idx += skip
			if idx >= t.m{
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