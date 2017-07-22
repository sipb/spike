package health

import (
	"github.com/sipb/spike/maglev"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
	"time"
)

// TODO use callback functions instead of depending on maglev

// A Server represents a backend service
type Server struct {
	health  bool
	service string
}

// Servers maps backends to services
type Servers map[string]*Server

// TODO write these functions as methods on Servers

// Serverstring returns the backends in servers
func Serverstring(servers Servers) []string {
	var backends []string
	for k := range servers {
		backends = append(backends, k)
	}
	return backends
}

// Addserver adds a server to the servers hash table
func Addserver(servers Servers, ip string, service string) {
	servers[ip] = &Server{false, service}
}

// Rmserver removes a server from the servers hash table
func Rmserver(servers Servers, ip string) {
	delete(servers, ip)
}

// Loopservers runs health checking asynchronously on all servers
func Loopservers(mm *maglev.Table, servers Servers,
	pollDelay time.Duration, timeout time.Duration) {
	for k := range servers {
		go loop(mm, servers, k, pollDelay, timeout)
	}
}

// Run health checking on a single server
func loop(mm *maglev.Table, servers Servers, ip string,
	pollDelay time.Duration, timeout time.Duration) {
	start := time.Now()

	// XXX unsynchronized writes are unsafe!

	for {
		if health(ip) {
			start = time.Now()
			if !servers[ip].health {
				log.Printf("server %v is healthy\n", ip)
				mm.Add(ip)
				servers[ip].health = true
			}
		}

		if servers[ip].health && time.Now().After(start.Add(timeout)) {
			log.Printf("server %v is down!\n", ip)
			servers[ip].health = false
			mm.Remove(ip)
		}

		time.Sleep(pollDelay)
	}
}

// Check health of server
func health(ip string) bool {
	// FIXME check errors

	resp, _ := http.Get(ip)
	bytes, _ := ioutil.ReadAll(resp.Body)

	resp.Body.Close()

	if resp == nil {
		return false
	}

	if strings.Contains(string(bytes), "healthy") {
		return true
	}

	return false
}
