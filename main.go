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

func main() {
	const lookupSizeM = 11

	backends := map[string]struct {
		ip   string
		quit chan struct{}
	}{
		"http://cheesy-fries.mit.edu/health":        {"1.2.3.4", nil},
		"http://strawberry-habanero.mit.edu/health": {"5.6.7.8", nil},
	}

	ips := make([]string, len(backends))
	i := 0
	for _, serviceInfo := range backends {
		ips[i] = serviceInfo.ip
		i++
	}

	// FIXME synchronize access to mm
	mm := maglev.New(ips, lookupSizeM)

	for service, serviceInfo := range backends {
		updates, quit := health.Check(service,
			100*time.Millisecond, 500*time.Millisecond)
		serviceInfo.quit = quit
		go func() {
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
		}()
	}

	ret := make(map[string]string)
	packets := []string{
		"19.168.124.100/572/81.9.179.69/80/4",
		"192.16.124.100/50270/81.209.179.69/80/6",
		"12.168.12.100/50268/81.209.179.69/80/6",
		"192.168.1.0/50266/81.209.179.69/80/6",
		"92.168.124.100/50264/81.209.179.69/80/6",
	}
	for i := 0; i < len(packets); i++ {
		serv := mm.Lookup(packets[i])
		ret[packets[i]] = serv
	}
	fmt.Printf("5-tuple to Server mapping:\n")
	for k, v := range ret {
		fmt.Printf("%v: %v\n", k, v)
	}

	// takes user input command to add or remove server
	scanner := bufio.NewScanner(os.Stdin)
	for {
		fmt.Print("> ")
		if !scanner.Scan() {
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
		default:
			fmt.Println("?")
			continue
		}
	}
}
