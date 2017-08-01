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
	"sync"
	"time"
)

type serviceInfo struct {
	ip   []byte
	quit chan<- struct{}
}

var globalMaglev *maglev.Table
var globalServices map[string]*serviceInfo
var globalServicesLock sync.Mutex

// Init initializes the spike health checker and consistent hashing modules.
//export Init
func Init() {
	// TODO dynamically rebuild maglev table with different M
	globalMaglev = maglev.New(maglev.SmallM)
	globalServices = make(map[string]*serviceInfo)
}

// AddBackend adds a new backend to the health checker.
//export AddBackend
func AddBackend(service string, ip []byte) {
	// FIXME make copies of passed-in data to avoid lua gc
	info := &serviceInfo{ip, nil}
	startChecker(globalMaglev, service, info)
	globalServicesLock.Lock()
	defer globalServicesLock.Unlock()
	globalServices[service] = info
}

// RemoveBackend removes a backend from the health checker.
//export RemoveBackend
func RemoveBackend(service string) {
	globalServicesLock.Lock()
	defer globalServicesLock.Unlock()
	info, ok := globalServices[service]
	if !ok {
		return
	}
	close(info.quit)
	delete(globalServices, service)
}

/*
TODO use separate arguments for the pieces - among other things we will
     need to discriminate on destination VIP
*/
// Lookup determines the backend associated with a five-tuple
//export Lookup
func Lookup(fiveTuple []byte) ([]byte, bool) {
	return globalMaglev.Lookup(fiveTuple)
}

func startChecker(mm *maglev.Table, service string, info *serviceInfo) {
	updates, quit := health.Check(service,
		100*time.Millisecond, 500*time.Millisecond)
	info.quit = quit
	go func() {
		for {
			up, ok := <-updates
			if !ok {
				return
			}
			if up {
				log.Printf("backend %v is healthy\n", service)
				mm.Add(info.ip)
			} else {
				log.Printf("backend %v is down!\n", service)
				mm.Remove(info.ip)
			}
		}
	}()
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
			info := &serviceInfo{net.ParseIP(words[2]), nil}
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
