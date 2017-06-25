package health

import (
	//import libraries
	"fmt"
	"net/http"
	"io/ioutil"
	"time"
	"../maglev"
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
func Addserver(servers map[string]*Server, ip string, service string) {
	servers[ip] = &Server{false, service}
}

//removes server from servers hash table
func Rmserver(servers map[string]*Server, ip string){
	delete(servers, ip)
}

//runs health checks on all servers
func Loopservers(mm *maglev.Table, servers map[string]*Server, num float64, timeout int){
	for k:= range servers{
		go loop(mm, servers, k, num, timeout)
	}
}

//runs health check on a single server
func loop(mm *maglev.Table, servers map[string]*Server, ip string, num float64, timeout int) {
	count := 0

	for {
		num := time.Duration(num)

		time.Sleep(num * time.Millisecond)
		//fmt.Println(ip, health(ip), "\n", count, servers)

		if health(ip) != true{
			count += 1
			fmt.Println(count)
		}

		if health(ip) == true {
			count = 0
			servers[ip].health = true
			mm.Add(ip)
		}

		if count >= timeout{ //change this later
			servers[ip].health = false
			mm.Remove(ip)
		}

	}
}

//checks health of server
func health(ip string) bool{
	resp, _ := http.Get(ip)
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
