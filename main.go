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

func main() {
	const lookupSizeM = 11

	backends := map[string]struct {
		ip   string
		quit chan struct{}
	}{
		"http://cheesy-fries.mit.edu/health":        {"1.2.3.4", nil},
		"http://strawberry-habanero.mit.edu/health": {"5.6.7.8", nil},
	}

	// FIXME synchronize access to mm
	mm := maglev.New(lookupSizeM)

	for service, serviceInfo := range backends {
		updates, quit := health.Check(service,
			100*time.Millisecond, 500*time.Millisecond)
		serviceInfo.quit = quit
		go func(service string, serviceInfo struct {
			ip   string
			quit chan struct{}
		}) {
			for {
				up, ok := <-updates
				if !ok {
					mm.Remove(serviceInfo.ip)
					return
				}
				if up {
					log.Printf("backend %v is healthy\n", service)
					mm.Add(serviceInfo.ip)
				} else {
					log.Printf("backend %v is down!\n", service)
					mm.Remove(serviceInfo.ip)
				}
			}
		}(service, serviceInfo)
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
			// TODO destroy checker
			fmt.Println("not implemented")
			continue
		case "addserver":
			// TODO spawn new checker
			fmt.Println("not implemented")
			continue
		case "lookup":
			fmt.Printf("5-tuple to Server mapping:\n")
			l := lookupPackets(mm, testPackets)
			fmt.Printf("5-tuple to Server mapping:\n")
			for _, p := range testPackets {
				fmt.Printf("%v: %v\n", p, l[p])
			}
			continue
		default:
			fmt.Println("?")
			continue
		}
	}
}
