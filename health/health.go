package health

import (
	//import libraries
	"fmt"
	"net/http"
	"io/ioutil"
	"time"
	"github.com/kkdai/maglev"
	"strings"
)

type Server struct {
	health bool
	service string
}

type Servers map[string]*Server


func Serverstring(servers map[string]*Server) []string{
	var names []string
	for k:= range servers{
		names = append(names, k)
	}
	return names
}

//adds server to servers hash table
func Addserver(servers map[string]*Server, url string, service string) {
	servers[url] = &Server{false, service}
}

//removes server from servers hash table
func Rmserver(servers map[string]*Server, url string){
	delete(servers, url)
}

//runs health checks on all servers
func Loopservers(mm *maglev.Maglev, servers map[string]*Server, num float64, timeout int){
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
