package tracking

import (
	"time"

	"github.com/sipb/spike/common"
)

type entry struct {
	backend *common.Backend
	expire  time.Time
}

type Cache struct {
	table  map[uint64]entry
	miss   func(uint64) (*common.Backend, bool)
	expiry time.Duration
}

func New(
	miss func(uint64) (*common.Backend, bool),
	expiry time.Duration,
) *Cache {
	return &Cache{
		table:  make(map[uint64]entry),
		miss:   miss,
		expiry: expiry,
	}
}

func (c *Cache) Lookup(key uint64) (*common.Backend, bool) {
	e, ok := c.table[key]
	if ok {
		if e.backend == nil || time.Now().After(e.expire) {
			ok = false
		} else {
			select {
			case <-e.backend.Unhealthy:
				ok = false
			default:
			}
		}
	}
	if !ok {
		e.backend, ok = c.miss(key)
		if !ok {
			delete(c.table, key)
			return nil, false
		}
	}
	e.expire = time.Now().Add(c.expiry)
	c.table[key] = e

	return e.backend, true
}
