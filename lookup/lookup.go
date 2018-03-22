package lookup

import (
	"net"
	"sync"
	"time"

	"github.com/sipb/spike/common"
	"github.com/sipb/spike/config"
	"github.com/sipb/spike/health"
	"github.com/sipb/spike/maglev"
	"github.com/sipb/spike/tracking"
)

const (
	healthDelay    time.Duration = 2 * time.Second
	healthTimeout                = 5 * time.Second
	httpTimeout                  = time.Second
	trackingExpiry               = 10 * time.Second
)

type backend struct {
	ip []byte

	health  health.Def
	checker health.Checker
}

// requires that info.ip and info.health are filled in already.
// healthy is true if the backend should be considered initially healthy.
func initHealth(m *maglev.Table, name string, info *backend,
	healthy bool) {
	backends := make(chan *common.Backend, 1)
	onUp := func() {
		down := make(chan struct{})
		backend := &common.Backend{
			IP:        info.ip,
			Unhealthy: down,
		}
		backends <- backend
		m.Add(backend)
	}
	onDown := func() {
		backend := <-backends
		close(backend.Unhealthy)
		m.Remove(backend)
	}

	if healthy {
		onUp()
	}

	info.checker = health.MakeChecker(info.health, healthy)
	go info.checker.Start()
	go health.Callback(info.checker, onUp, onDown)
}

type pool struct {
	backends map[string]*backend
	table    *maglev.Table
}

type T struct {
	poolsLock sync.RWMutex
	pools     map[[16]byte]pool
	tracker   *tracking.Cache
}

func New() *T {
	s := new(T)
	s.tracker = tracking.New(
		func(f common.FiveTuple) (*common.Backend, bool) {
			s.poolsLock.RLock()
			pool, ok := s.pools[f.Dst_ip]
			s.poolsLock.RUnlock()
			if !ok {
				return nil, false
			}
			return pool.table.Lookup5(f)
		}, trackingExpiry)
	return s
}

func (s *T) Reconfig(config *config.T) {
	s.poolsLock.Lock()
	defer s.poolsLock.Unlock()

	newPools := make(map[[16]byte]pool)
	for _, p := range config.Pools {
		vip := common.AddrTo16(net.ParseIP(p.VIP))
		backends := make(map[string]*backend)
		oldPool := s.pools[vip]
		table := maglev.New(p.MaglevSize)
		for _, b := range p.Backends {
			info := backend{
				ip: net.ParseIP(b.IP),
				health: health.Def{
					Type:        b.HealthCheck.Type,
					Delay:       healthDelay,
					Timeout:     healthTimeout,
					HTTPAddr:    b.HealthCheck.HTTPAddr,
					HTTPTimeout: httpTimeout,
				},
			}

			backends[b.Name] = &info

			oldBackend, ok := oldPool.backends[b.Name]
			healthy := b.HealthCheck.Healthy
			if ok && oldBackend.health == info.health {
				healthy = oldBackend.checker.Stop()
				// prevent it from being stopped again later
				delete(oldPool.backends, b.Name)
			}
			initHealth(table, b.Name, &info, healthy)
		}
		newPools[vip] = pool{
			backends: backends,
			table:    table,
		}
	}

	oldPools := s.pools
	s.pools = newPools

	go func() {
		// stop checkers not in use
		for _, pool := range oldPools {
			for _, b := range pool.backends {
				b.checker.Stop()
			}
		}
	}()
}

func (s *T) Lookup(f common.FiveTuple) (*common.Backend, bool) {
	return s.tracker.Lookup(f)
}
