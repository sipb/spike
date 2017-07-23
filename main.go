package main

import (
	"bufio"
	"fmt"
	"github.com/sipb/spike/health"
	"github.com/sipb/spike/maglev"
	"log"
	"os"
	"strings"
	"time"
)

func lookupPackets(mm *maglev.Table, packets []string) map[string]string {
	ret := make(map[string]string)
	for _, p := range packets {
		if serv, ok := mm.Lookup(p); ok {
			ret[p] = serv
		} else {
			ret[p] = "(no value)"
		}
	}
	return ret
}

type serviceInfo struct {
	ip   string
	quit chan struct{}
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
		"http://cheesy-fries.mit.edu/health":        &serviceInfo{"1.2.3.4", nil},
		"http://strawberry-habanero.mit.edu/health": &serviceInfo{"5.6.7.8", nil},
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
			info := &serviceInfo{words[2], nil}
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
