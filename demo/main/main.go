package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/sipb/spike/common"
	"github.com/sipb/spike/config"
	"github.com/sipb/spike/lookup"
)

func main() {
	c, err := config.Read("config.yaml")
	if err != nil {
		log.Fatal(err)
	}

	s := lookup.New()
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

func lookupPackets(
	s *lookup.T,
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
