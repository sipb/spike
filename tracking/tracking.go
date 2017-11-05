package tracking

import (
	"time"

	"github.com/sipb/spike/common"
)

type entry struct {
	backend *common.Backend
	expire  time.Time
}

// Cache is a connection-tracking table.  It lazily evicts entries when
// the backend becomes unhealthy or when the entry expires by not been
// accessed.
//
// Cache is not thread-safe.
type Cache struct {
	table  map[common.FiveTuple]entry
	miss   func(common.FiveTuple) (*common.Backend, bool)
	expiry time.Duration
}

// New constructs a new connection-tracking table which caches the given
// function.
func New(
	miss func(common.FiveTuple) (*common.Backend, bool),
	expiry time.Duration,
) *Cache {
	return &Cache{
		table:  make(map[common.FiveTuple]entry),
		miss:   miss,
		expiry: expiry,
	}
}

// Lookup returns the backend associated with the given key.  If the
// cached backend is unhealthy, or the key is not cached, it retrieves a
// backend from the underlying function.  Lookup returns false if no
// backend is available.
func (c *Cache) Lookup(key common.FiveTuple) (*common.Backend, bool) {
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
