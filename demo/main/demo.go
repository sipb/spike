package main

import "C"

import (
	"bufio"
	"fmt"
	"github.com/sipb/spike/health"
	"github.com/sipb/spike/maglev"
	"log"
	"net"
	"os"
	"strings"
	"time"
)

type serviceInfo struct {
	ip   []byte
	quit chan<- struct{}
}

func startChecker(mm *maglev.Table, service string, info *serviceInfo) {
	info.quit = health.CheckFun(service,
		func() {
			log.Printf("backend %v is healthy\n", service)
			mm.Add(info.ip)
		},
		func() {
			log.Printf("backend %v is down\n", service)
			mm.Remove(info.ip)
		},
		100*time.Millisecond, 500*time.Millisecond)
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

	for service, info := range backends {
		startChecker(mm, service, info)
	}

	testPackets := []string{
		"19.168.124.100/572/81.9.179.69/80/4",
		"192.16.124.100/50270/81.209.179.69/80/6",
		"12.168.12.100/50268/81.209.179.69/80/6",
		"192.168.1.0/50266/81.209.179.69/80/6",
		"92.168.124.100/50264/81.209.179.69/80/6",
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
			l := lookupPackets(mm, testPackets)
			fmt.Printf("5-tuple to Server mapping:\n")
			for _, p := range testPackets {
				fmt.Printf("%v: %v\n", p, l[p])
			}
		default:
			fmt.Println("?")
		}
	}
}

func lookupPackets(mm *maglev.Table, packets []string) map[string][]byte {
	ret := make(map[string][]byte)
	for _, p := range packets {
		if serv, ok := mm.Lookup([]byte(p)); ok {
			ret[p] = serv
		} else {
			ret[p] = nil
		}
	}
	return ret
}
