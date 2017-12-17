package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"os"
	"strings"
	"time"

	"github.com/sipb/spike/common"
	"github.com/sipb/spike/health"
	"github.com/sipb/spike/maglev"
	"github.com/sipb/spike/tracking"
)

type backendInfo struct {
	ip            []byte
	quit          chan<- struct{}
	healthService string
}

func startChecker(mm *maglev.Table, name string, info *backendInfo) {
	quit := make(chan struct{})
	info.quit = quit
	backends := make(chan *common.Backend, 1)

	health.CheckFun(func() bool {
		if info.healthService == "" {
			return true
		}
		return health.HTTP(info.healthService, 2*time.Second)
	},
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
		time.Second, 5*time.Second, quit)
}

func main() {
	const lookupSizeM = 11
	type pool struct{
		info  map[string]*backendInfo
		table *maglev.Table
	}

	pools := make(map[[16]byte]pool)
	var ip0, ip1 [16]byte
	for i, x := range net.ParseIP("18.18.18.18").To16() {
		ip0[i] = x
	}
	for i, x := range net.ParseIP("18.18.18.19").To16() {
		ip1[i] = x
	}
	pools[ip0] = pool{
		map[string]*backendInfo{
			"cheesy-fries": &backendInfo{
				[]byte{1, 2, 3, 4}, nil, "http://cheesy-fries.mit.edu/health"},
			"strawberry-habanero": &backendInfo{
				[]byte{5, 6, 7, 8}, nil, "http://strawberry-habanero.mit.edu/health"},
			"powerful-motor": &backendInfo{
				[]byte{9, 10, 11, 12}, nil, ""},
		},
		maglev.New(lookupSizeM),
	}
	pools[ip1] = pool{
		map[string]*backendInfo{
			"godel": &backendInfo{
				[]byte{1, 0, 0, 1}, nil, ""},
			"lob": &backendInfo{
				[]byte{1, 1, 0, 0}, nil, ""},
		},
		maglev.New(lookupSizeM),
	}

	tt := tracking.New(func(f common.FiveTuple) (*common.Backend, bool) {
		z, ok := pools[f.Dst_ip]
		if !ok {
			return nil, false
		}
		return z.table.Lookup5(f)
	}, 10*time.Second)

	for _, pool := range pools {
		for name, info := range pool.info {
			startChecker(pool.table, name, info)
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
			close(info.quit)
			delete(p.info, words[2])
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
				p = pool{
					make(map[string]*backendInfo),
					maglev.New(lookupSizeM),
				}
				pools[paddr_a] = p
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
			info := &backendInfo{addr, nil, words[4]}
			startChecker(p.table, words[1], info)
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
					if info.healthService == "" {
						fmt.Print(" (health mocked)")
					} else {
						fmt.Printf("; healthcheck %v", info.healthService)
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
