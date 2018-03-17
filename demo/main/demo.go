package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"os"
	"strings"
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

type backendInfo struct {
	ip []byte

	health  health.Def
	checker health.Checker
}

type pool struct {
	backends map[string]*backendInfo
	table    *maglev.Table
}

type Spike struct {
	poolsLock sync.RWMutex
	pools     map[[16]byte]pool
	tracker   *tracking.Cache
}

// requires that info.ip and info.health are filled in already
func initHealth(mm *maglev.Table, name string, info *backendInfo,
	healthy bool) {
	backends := make(chan *common.Backend, 1)
	onUp := func() {
		log.Printf("backend %v is healthy\n", name)
		down := make(chan struct{})
		backend := &common.Backend{
			IP:        info.ip,
			Unhealthy: down,
		}
		backends <- backend
		mm.Add(backend)
	}
	onDown := func() {
		log.Printf("backend %v is down\n", name)
		backend := <-backends
		close(backend.Unhealthy)
		mm.Remove(backend)
	}

	if healthy {
		onUp()
	}

	info.checker = health.MakeChecker(info.health, healthy)
	go info.checker.Start()
	go health.Callback(info.checker, onUp, onDown)
}

func NewSpike() *Spike {
	s := new(Spike)
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

func main() {
	c, err := config.Read("config.yaml")
	if err != nil {
		log.Fatal(err)
	}

	s := NewSpike()
	s.Reconfig(c)

	testPackets := make([]common.FiveTuple, 0, 10)
	for i := byte(0); i < 10; i++ {
		testPackets = append(testPackets,
			common.NewFiveTuple(
				[]byte{10*i + 0, 10*i + 1, 10*i + 2, 10*i + 3},
				[]byte{18, 18, 18, 18 + i%3},
				uint16(10*i+8), uint16(10*i+9), common.L3_IPV4))
	}

	// takes user input command to add or remove server
	scanner := bufio.NewScanner(os.Stdin)
	for {
		fmt.Print("> ")
		if !scanner.Scan() {
			fmt.Println()
			break
		}
		input := scanner.Text()
		words := strings.Fields(input)

		if len(words) < 1 {
			continue
		}

		switch words[0] {
		case "help":
			fmt.Println("commands:")
			fmt.Println("help")
			fmt.Println("lookup")
			fmt.Println("reconfig")
			fmt.Println("quit")
		case "lookup":
			l := lookupPackets(s, testPackets)
			fmt.Printf("5-tuple to Server mapping:\n")
			for _, p := range testPackets {
				fmt.Printf("%v: %v\n", p, l[p])
			}
		case "reconfig":
			if len(words) != 2 {
				fmt.Println("usage: reconfig <filename>")
				continue
			}
			c, err := config.Read(words[1])
			if err != nil {
				fmt.Printf("error reading config: %v", err)
			}
			s.Reconfig(c)
			fmt.Println("ok")
		case "quit":
			return
		default:
			fmt.Println("?")
		}
	}
}

func (s *Spike) Reconfig(config *config.T) {
	s.poolsLock.Lock()
	defer s.poolsLock.Unlock()

	newPools := make(map[[16]byte]pool)
	for _, p := range config.Pools {
		vip := common.AddrTo16(net.ParseIP(p.VIP))
		backends := make(map[string]*backendInfo)
		oldPool := s.pools[vip]
		table := maglev.New(p.MaglevSize)
		for _, b := range p.Backends {
			info := backendInfo{
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

func lookupPackets(
	s *Spike,
	packets []common.FiveTuple,
) map[common.FiveTuple][]byte {
	ret := make(map[common.FiveTuple][]byte)
	for _, p := range packets {
		if serv, ok := s.Lookup(p); ok {
			ret[p] = serv.IP
		} else {
			ret[p] = nil
		}
	}
	return ret
}

func (s *Spike) Lookup(f common.FiveTuple) (*common.Backend, bool) {
	return s.tracker.Lookup(f)
}
