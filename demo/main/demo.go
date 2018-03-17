package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"os"
	"strconv"
	"strings"
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

// requires that info.ip and info.health are filled in already
func initHealth(mm *maglev.Table, name string, info *backendInfo) {
	backends := make(chan *common.Backend, 1)

	info.checker = health.MakeChecker(info.health)
	go info.checker.Start()
	go health.Callback(info.checker,
		func() {
			log.Printf("backend %v is healthy\n", name)
			down := make(chan struct{})
			backend := &common.Backend{
				IP:        info.ip,
				Unhealthy: down,
			}
			backends <- backend
			mm.Add(backend)
		},
		func() {
			log.Printf("backend %v is down\n", name)
			backend := <-backends
			close(backend.Unhealthy)
			mm.Remove(backend)
		},
	)
}

func main() {
	type pool struct {
		info  map[string]*backendInfo
		table *maglev.Table
	}

	config, err := config.Read("config.yaml")
	if err != nil {
		log.Fatal(err)
	}
	pools := make(map[[16]byte]pool)
	for _, p := range config.Pools {
		var vip [16]byte
		for i, x := range net.ParseIP(p.VIP).To16() {
			vip[i] = x
		}
		backends := make(map[string]*backendInfo)
		for _, b := range p.Backends {
			backends[b.Name] = &backendInfo{
				ip: net.ParseIP(b.IP),
				health: health.Def{
					Type: b.HealthCheck.Type,

					Delay:   healthDelay,
					Timeout: healthTimeout,

					HTTPAddr:    b.HealthCheck.HTTPAddr,
					HTTPTimeout: httpTimeout,
				},
			}
		}
		pools[vip] = pool{
			backends,
			maglev.New(p.MaglevSize),
		}
	}

	tt := tracking.New(func(f common.FiveTuple) (*common.Backend, bool) {
		z, ok := pools[f.Dst_ip]
		if !ok {
			return nil, false
		}
		return z.table.Lookup5(f)
	}, trackingExpiry)

	for _, pool := range pools {
		for name, info := range pool.info {
			initHealth(pool.table, name, info)
		}
	}

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
			fmt.Println("addpool pool size")
			fmt.Println("addbackend pool name IP [healthService]")
			fmt.Println("rmbackend pool name")
			fmt.Println("list")
			fmt.Println("lookup")
			fmt.Println("quit")
		case "rmbackend":
			if len(words) != 3 {
				fmt.Println("?")
				continue
			}
			paddr := net.ParseIP(words[1]).To16()
			if paddr == nil {
				fmt.Println("invalid pool address")
				continue
			}
			p, ok := pools[common.AddrTo16(paddr)]
			if !ok {
				fmt.Println("no such pool")
				continue
			}
			info, ok := p.info[words[2]]
			if !ok {
				fmt.Println("no such backend")
				continue
			}
			info.checker.Stop()
			delete(p.info, words[2])
		case "addpool":
			if len(words) != 3 {
				fmt.Println("?")
				continue
			}
			paddr := net.ParseIP(words[1]).To16()
			if paddr == nil {
				fmt.Println("invalid pool address")
				continue
			}
			paddr_a := common.AddrTo16(paddr)
			size, err := strconv.Atoi(words[2])
			if err != nil {
				fmt.Println(err)
				continue
			}
			p := pool{
				make(map[string]*backendInfo),
				maglev.New(uint64(size)),
			}
			if _, ok := pools[paddr_a]; ok {
				fmt.Println("pool exists")
				continue
			}
			pools[paddr_a] = p
		case "addbackend":
			if len(words) == 4 {
				words = append(words, "")
			}
			if len(words) != 5 {
				fmt.Println("?")
				continue
			}
			paddr := net.ParseIP(words[1]).To16()
			if paddr == nil {
				fmt.Println("invalid pool address")
				continue
			}
			paddr_a := common.AddrTo16(paddr)
			p, ok := pools[paddr_a]
			if !ok {
				fmt.Println("no such pool")
				continue
			}
			if _, ok := p.info[words[2]]; ok {
				fmt.Println("backend already exists")
				continue
			}
			addr := net.ParseIP(words[3])
			if addr == nil {
				fmt.Println("invalid backend address")
				continue
			}
			var d health.Def
			if words[4] == "" {
				d.Type = "none"
			} else {
				d.Type = "http"
			}
			d.Delay = healthDelay
			d.Timeout = healthTimeout
			d.HTTPAddr = words[4]
			d.HTTPTimeout = httpTimeout
			info := &backendInfo{
				ip:     addr,
				health: d,
			}
			initHealth(p.table, words[2], info)
			p.info[words[2]] = info
		case "lookup":
			l := lookupPackets(tt, testPackets)
			fmt.Printf("5-tuple to Server mapping:\n")
			for _, p := range testPackets {
				fmt.Printf("%v: %v\n", p, l[p])
			}
		case "list":
			for paddr, p := range pools {
				fmt.Printf("VIP %v:\n", paddr)
				for backend, info := range p.info {
					fmt.Printf("backend %v: IP %v", backend, info.ip)
					switch info.health.Type {
					case "none":
						fmt.Print(" (health mocked)")
					case "http":
						fmt.Printf("; healthcheck %v", info.health.HTTPAddr)
					default:
						panic("unknown health type")
					}
					fmt.Println()
				}
				fmt.Println()
			}
		case "quit":
			return
		default:
			fmt.Println("?")
		}
	}
}

func lookupPackets(
	tt *tracking.Cache,
	packets []common.FiveTuple,
) map[common.FiveTuple][]byte {
	ret := make(map[common.FiveTuple][]byte)
	for _, p := range packets {
		if serv, ok := tt.Lookup(p); ok {
			ret[p] = serv.IP
		} else {
			ret[p] = nil
		}
	}
	return ret
}
