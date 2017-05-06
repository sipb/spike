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
	"errors"
)

type Server struct {
	health bool
	service string
}

type Servers map[string]*Server

//adds server to servers hash table
func addserver(servers map[string]*Server, url string, service string) {
	servers[url] = &Server{false, service}
}

//removes server from servers hash table
func rmserver(servers map[string]*Server, url string){
	delete(servers, url)
}

//runs health checks on all servers
func loopservers(mm *Maglev, servers map[string]*Server, num float64, timeout int){
	for k:= range servers{
		go loop(mm, servers, k, num, timeout)
	}
}

//runs health check on a single server
func loop(mm *Maglev, servers map[string]*Server, url string, num float64, timeout int) {
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

const BlockSize = 64

// Hash returns the 64-bit SipHash-2-4 of the given byte slice with two 64-bit
// parts of 128-bit key: k0 and k1.
func Hash(k0, k1 uint64, p []byte) uint64 {
	// Initialization.
	v0 := k0 ^ 0x736f6d6570736575
	v1 := k1 ^ 0x646f72616e646f6d
	v2 := k0 ^ 0x6c7967656e657261
	v3 := k1 ^ 0x7465646279746573
	t := uint64(len(p)) << 56

	// Compression.
	for len(p) >= BlockSize {
		m := uint64(p[0]) | uint64(p[1])<<8 | uint64(p[2])<<16 | uint64(p[3])<<24 |
			uint64(p[4])<<32 | uint64(p[5])<<40 | uint64(p[6])<<48 | uint64(p[7])<<56
		v3 ^= m

		// Round 1.
		v0 += v1
		v1 = v1<<13 | v1>>(64-13)
		v1 ^= v0
		v0 = v0<<32 | v0>>(64-32)

		v2 += v3
		v3 = v3<<16 | v3>>(64-16)
		v3 ^= v2

		v0 += v3
		v3 = v3<<21 | v3>>(64-21)
		v3 ^= v0

		v2 += v1
		v1 = v1<<17 | v1>>(64-17)
		v1 ^= v2
		v2 = v2<<32 | v2>>(64-32)

		// Round 2.
		v0 += v1
		v1 = v1<<13 | v1>>(64-13)
		v1 ^= v0
		v0 = v0<<32 | v0>>(64-32)

		v2 += v3
		v3 = v3<<16 | v3>>(64-16)
		v3 ^= v2

		v0 += v3
		v3 = v3<<21 | v3>>(64-21)
		v3 ^= v0

		v2 += v1
		v1 = v1<<17 | v1>>(64-17)
		v1 ^= v2
		v2 = v2<<32 | v2>>(64-32)

		v0 ^= m
		p = p[BlockSize:]
	}

	// Compress last block.
	switch len(p) {
	case 7:
		t |= uint64(p[6]) << 48
		fallthrough
	case 6:
		t |= uint64(p[5]) << 40
		fallthrough
	case 5:
		t |= uint64(p[4]) << 32
		fallthrough
	case 4:
		t |= uint64(p[3]) << 24
		fallthrough
	case 3:
		t |= uint64(p[2]) << 16
		fallthrough
	case 2:
		t |= uint64(p[1]) << 8
		fallthrough
	case 1:
		t |= uint64(p[0])
	}

	v3 ^= t

	// Round 1.
	v0 += v1
	v1 = v1<<13 | v1>>(64-13)
	v1 ^= v0
	v0 = v0<<32 | v0>>(64-32)

	v2 += v3
	v3 = v3<<16 | v3>>(64-16)
	v3 ^= v2

	v0 += v3
	v3 = v3<<21 | v3>>(64-21)
	v3 ^= v0

	v2 += v1
	v1 = v1<<17 | v1>>(64-17)
	v1 ^= v2
	v2 = v2<<32 | v2>>(64-32)

	// Round 2.
	v0 += v1
	v1 = v1<<13 | v1>>(64-13)
	v1 ^= v0
	v0 = v0<<32 | v0>>(64-32)

	v2 += v3
	v3 = v3<<16 | v3>>(64-16)
	v3 ^= v2

	v0 += v3
	v3 = v3<<21 | v3>>(64-21)
	v3 ^= v0

	v2 += v1
	v1 = v1<<17 | v1>>(64-17)
	v1 ^= v2
	v2 = v2<<32 | v2>>(64-32)

	v0 ^= t

	// Finalization.
	v2 ^= 0xff

	// Round 1.
	v0 += v1
	v1 = v1<<13 | v1>>(64-13)
	v1 ^= v0
	v0 = v0<<32 | v0>>(64-32)

	v2 += v3
	v3 = v3<<16 | v3>>(64-16)
	v3 ^= v2

	v0 += v3
	v3 = v3<<21 | v3>>(64-21)
	v3 ^= v0

	v2 += v1
	v1 = v1<<17 | v1>>(64-17)
	v1 ^= v2
	v2 = v2<<32 | v2>>(64-32)

	// Round 2.
	v0 += v1
	v1 = v1<<13 | v1>>(64-13)
	v1 ^= v0
	v0 = v0<<32 | v0>>(64-32)

	v2 += v3
	v3 = v3<<16 | v3>>(64-16)
	v3 ^= v2

	v0 += v3
	v3 = v3<<21 | v3>>(64-21)
	v3 ^= v0

	v2 += v1
	v1 = v1<<17 | v1>>(64-17)
	v1 ^= v2
	v2 = v2<<32 | v2>>(64-32)

	// Round 3.
	v0 += v1
	v1 = v1<<13 | v1>>(64-13)
	v1 ^= v0
	v0 = v0<<32 | v0>>(64-32)

	v2 += v3
	v3 = v3<<16 | v3>>(64-16)
	v3 ^= v2

	v0 += v3
	v3 = v3<<21 | v3>>(64-21)
	v3 ^= v0

	v2 += v1
	v1 = v1<<17 | v1>>(64-17)
	v1 ^= v2
	v2 = v2<<32 | v2>>(64-32)

	// Round 4.
	v0 += v1
	v1 = v1<<13 | v1>>(64-13)
	v1 ^= v0
	v0 = v0<<32 | v0>>(64-32)

	v2 += v3
	v3 = v3<<16 | v3>>(64-16)
	v3 ^= v2

	v0 += v3
	v3 = v3<<21 | v3>>(64-21)
	v3 ^= v0

	v2 += v1
	v1 = v1<<17 | v1>>(64-17)
	v1 ^= v2
	v2 = v2<<32 | v2>>(64-32)

	return v0 ^ v1 ^ v2 ^ v3
}

const (
	bigM uint64 = 65537
)

//Maglev :
type Maglev struct {
	n           uint64 //size of VIP backends
	m           uint64 //sie of the lookup table
	permutation [][]uint64
	lookup      []int64
	nodeList    []string
}

//NewMaglev :
func NewMaglev(backends []string, m uint64) *Maglev {
	mag := &Maglev{n: uint64(len(backends)), m: m}
	mag.nodeList = backends
	mag.generatePopulation()
	mag.populate()
	return mag
}

//Add : Return nil if add success, otherwise return error
func (m *Maglev) Add(backend string) error {
	for _, v := range m.nodeList {
		if v == backend {
			return errors.New("Exist already")
		}
	}

	m.nodeList = append(m.nodeList, backend)
	m.n = uint64(len(m.nodeList))
	m.generatePopulation()
	m.populate()
	return nil
}

//Remove :
func (m *Maglev) Remove(backend string) error {
	notFound := true
	for _, v := range m.nodeList {
		if v == backend {
			notFound = false
		}
	}
	if notFound {
		return errors.New("Not found")
	}

	for i, v := range m.nodeList {
		if v == backend {
			m.nodeList = append(m.nodeList[:i], m.nodeList[i+1:]...)
			break
		}
	}

	m.n = uint64(len(m.nodeList))
	m.generatePopulation()
	m.populate()
	return nil
}

//Get :Get node name by object string.
func (m *Maglev) Get(obj string) (string, error) {
	if len(m.nodeList) == 0 {
		return "", errors.New("Empty")
	}
	key := m.hashKey(obj)
	return m.nodeList[m.lookup[key%m.m]], nil
}

func (m *Maglev) hashKey(obj string) uint64 {
	return Hash(0xdeadbabe, 0, []byte(obj))
}

func (m *Maglev) generatePopulation() {
	if len(m.nodeList) == 0 {
		return
	}

	for i := 0; i < len(m.nodeList); i++ {
		bData := []byte(m.nodeList[i])

		offset := Hash(0xdeadbabe, 0, bData) % m.m
		skip := (Hash(0xdeadbeef, 0, bData) % (m.m - 1)) + 1

		iRow := make([]uint64, m.m)
		var j uint64
		for j = 0; j < m.m; j++ {
			iRow[j] = (offset + uint64(j)*skip) % m.m
		}

		m.permutation = append(m.permutation, iRow)
	}
}

func (m *Maglev) populate() {
	if len(m.nodeList) == 0 {
		return
	}

	var i, j uint64
	next := make([]uint64, m.n)
	entry := make([]int64, m.m)
	for j = 0; j < m.m; j++ {
		entry[j] = -1
	}

	var n uint64

	for { //true
		for i = 0; i < m.n; i++ {
			c := m.permutation[i][next[i]]
			for entry[c] >= 0 {
				next[i] = next[i] + 1
				c = m.permutation[i][next[i]]
			}

			entry[c] = int64(i)
			next[i] = next[i] + 1
			n++

			if n == m.m {
				m.lookup = entry
				return
			}
		}

	}

}

func serverstring(servers map[string]*Server) []string{
	var names []string
	for k:= range servers{
		names = append(names, k)
	}
	return names
}

const sizeN = 2
const lookupSizeM = 13

func main() {
	servers := make(Servers)

	addserver(servers, "http://cheesy-fries.mit.edu/health", "service")
	addserver(servers, "http://strawberry-habanero.mit.edu/health", "service")

	names := serverstring(servers)
  	mm := NewMaglev(names, lookupSizeM)

	loopservers(mm, servers, 100, 500)

    fmt.Printf("%v\n", mm.lookup)
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