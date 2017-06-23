package main

import (
	//import libraries
	"fmt"
	"net/http"
	"io/ioutil"
	"strings"
	"time"
	"bufio"
	"os"
	"github.com/kkdai/maglev"
)

type Server struct {
	health bool
	service string
}

type Servers map[string]*Server

func serverstring(servers map[string]*Server) []string{
	var names []string
	for k:= range servers{
		names = append(names, k)
	}
	return names
}

const lookupSizeM = 11

func main() {
	servers := make(Servers)

	addserver(servers, "http://cheesy-fries.mit.edu/health", "service")
	addserver(servers, "http://strawberry-habanero.mit.edu/health", "service")


	names := serverstring(servers)
  	mm := maglev.NewMaglev(names, lookupSizeM)

	loopservers(mm, servers, 100, 500)

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
			rmserver(servers, words[1])
			fmt.Println(servers)
		}

		if strings.Contains(input, "addserver"){
			addserver(servers, words[1], words[2])
			fmt.Println(servers)
		}
	}
}


//adds server to servers hash table
func addserver(servers map[string]*Server, url string, service string) {
	servers[url] = &Server{false, service}
}

//removes server from servers hash table
func rmserver(servers map[string]*Server, url string){
	delete(servers, url)
}

//runs health checks on all servers
func loopservers(mm *maglev.Maglev, servers map[string]*Server, num float64, timeout int){
	for k:= range servers{
		go loop(mm, servers, k, num, timeout)
	}
}

//runs health check on a single server
func loop(mm *maglev.Maglev, servers map[string]*Server, url string, num float64, timeout int) {
	count := 0
	boo := true

	for boo{
		num := time.Duration(num)

		time.Sleep(num * time.Millisecond)
		//fmt.Println(url, health(url), "\n", count, servers)

		if health(url) != true{
			count += 1
			fmt.Println(count)
		}

		if health(url) == true {
			count = 0
			servers[url].health = true
			mm.Add(url)
		}

		if count >= timeout{ //change this later
			servers[url].health = false
			mm.Remove(url)
		}

	}
}

//checks health of server
func health(url string) bool{
	resp, _ := http.Get(url)
	bytes, _ := ioutil.ReadAll(resp.Body)

	resp.Body.Close()

	if resp == nil {
		return false
	}

	if strings.Contains(string(bytes),"healthy") {
		return true
	}

	return false
}
