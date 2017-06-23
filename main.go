package main

import (
	"./health"
	"github.com/kkdai/maglev"
	"bufio"
	"os"
	"fmt"
	"strings"
	)

func main() {
	const lookupSizeM = 11
	servers := make(health.Servers)

	health.Addserver(servers, "http://cheesy-fries.mit.edu/health", "service")
	health.Addserver(servers, "http://strawberry-habanero.mit.edu/health", "service")

	names := health.Serverstring(servers)
  	mm := maglev.NewMaglev(names, lookupSizeM)

	health.Loopservers(mm, servers, 100, 500)

    ret := make(map[string]string)
    packets := []string{"19.168.124.100/572/81.9.179.69/80/4", "192.16.124.100/50270/81.209.179.69/80/6", "12.168.12.100/50268/81.209.179.69/80/6", "192.168.1.0/50266/81.209.179.69/80/6", "92.168.124.100/50264/81.209.179.69/80/6"}
    for i := 0; i < len(packets); i++ {
      serv, _ := mm.Get(packets[i])
      ret[packets[i]] = serv
    }
    fmt.Printf("5-tuple to Server mapping:\n")
    for k, v := range ret {
      fmt.Printf("%v: %v\n", k, v)
    }

	//takes user input command to add or remove server
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		var input = scanner.Text()
		fmt.Println("Executing: ",input)
		words := strings.Fields(input)

		//add server works but rm server makes loopserver in line 27 crash
		//need to implement channel...?
		if strings.Contains(input, "rmserver"){
			health.Rmserver(servers, words[1])
			fmt.Println(servers)
		}

		if strings.Contains(input, "addserver"){
			health.Addserver(servers, words[1], words[2])
			fmt.Println(servers)
		}
	}
}
