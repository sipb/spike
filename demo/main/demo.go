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

type serviceInfo struct {
	ip   []byte
	quit chan<- struct{}
}

func startChecker(mm *maglev.Table, service string, info *serviceInfo) {
	quit := make(chan struct{})
	info.quit = quit
	backends := make(chan *common.Backend, 1)

	health.CheckFun(service, func() bool {
		return health.HTTP(service, 2*time.Second)
	},
		func() {
			log.Printf("backend %v is healthy\n", service)
			down := make(chan struct{})
			backend := &common.Backend{
				IP:        info.ip,
				Unhealthy: down,
			}
			backends <- backend
			mm.Add(backend)
		},
		func() {
			log.Printf("backend %v is down\n", service)
			backend := <-backends
			close(backend.Unhealthy)
			mm.Remove(backend)
		},
		time.Second, 5*time.Second, quit)
}

func main() {
	const lookupSizeM = 11

	backends := map[string]*serviceInfo{
		"http://cheesy-fries.mit.edu/health": &serviceInfo{
			[]byte{1, 2, 3, 4}, nil},
		"http://strawberry-habanero.mit.edu/health": &serviceInfo{
			[]byte{5, 6, 7, 8}, nil},
	}

	mm := maglev.New(lookupSizeM)
	tt := tracking.New(mm.Lookup5, 10*time.Second)

	for service, info := range backends {
		startChecker(mm, service, info)
	}

	testPackets := make([]common.FiveTuple, 0, 10)
	for i := byte(0); i < 10; i++ {
		testPackets = append(testPackets,
			common.NewFiveTuple(
				[]byte{10*i + 0, 10*i + 1, 10*i + 2, 10*i + 3},
				[]byte{10*i + 4, 10*i + 5, 10*i + 6, 10*i + 7},
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
			fmt.Println("addserver <service> <IP>")
			fmt.Println("rmserver <service>")
			fmt.Println("lookup")
		case "rmserver":
			if len(words) != 2 {
				fmt.Println("?")
				continue
			}
			info, ok := backends[words[1]]
			if !ok {
				fmt.Println("no such backend")
				continue
			}
			close(info.quit)
			delete(backends, words[1])
		case "addserver":
			if len(words) != 3 {
				fmt.Println("?")
				continue
			}
			if _, ok := backends[words[1]]; ok {
				fmt.Println("backend already exists")
				continue
			}
			addr := net.ParseIP(words[2]).To4()
			if addr == nil {
				fmt.Println("not an IPv4 address")
				continue
			}
			info := &serviceInfo{addr, nil}
			startChecker(mm, words[1], info)
			backends[words[1]] = info
		case "lookup":
			l := lookupPackets(tt, testPackets)
			fmt.Printf("5-tuple to Server mapping:\n")
			for _, p := range testPackets {
				fmt.Printf("%v: %v\n", p, l[p])
			}
		default:
			fmt.Println("?")
		}
	}
}

func lookupPackets(tt *tracking.Cache, packets []common.FiveTuple) map[common.FiveTuple][]byte {
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
