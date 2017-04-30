package main

import (
	//import libraries
	"fmt"
	"net/http"
	"io/ioutil"
	"strings"
	"time"
)

type Server struct {
	health bool
	service string
}

type Servers map[string]*Server

func main() {
	servers := make(Servers)

	addserver(servers, "http://cheesy-fries.mit.edu/health", "service")
	addserver(servers, "http://strawberry-habanero.mit.edu/health", "service")

	loopservers(servers, 100, 500)
	loop(servers, "http://strawberry-habanero.mit.edu/health", 100, 500)
}


//adds server to servers hash table
func addserver(servers map[string]*Server, url string, service string) {
	servers[url] = &Server{false, service}
}

//removes server from servers hash table
func rmserver(servers map[string]*Server, url string){
	delete(servers, url)
}

func loopservers(servers map[string]*Server, num float64, timeout int){
	for k:= range servers{
		go loop(servers, k, num, timeout)
	}
}

func loop(servers map[string]*Server, url string, num float64, timeout int) {
	count := 0
	boo := true

	for boo{
		num := time.Duration(num)

		time.Sleep(num * time.Millisecond)
		fmt.Println(url, health(url), "\n", count, servers)

		if health(url) != true{
			count += 1
			fmt.Println(count)
		}

		if health(url) == true {
			count = 0
			servers[url].health = true
		}

		if count >= timeout{ //change this later
			servers[url].health = false
		}

	}
}

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