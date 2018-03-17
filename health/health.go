package health

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"
)

type Def struct {
	// valid types are "mock", "http"
	Type string

	// ignored for type "mock"
	Delay   time.Duration
	Timeout time.Duration

	// only for type "http"
	HTTPAddr    string
	HTTPTimeout time.Duration
}

func MakeChecker(d Def, healthy bool) Checker {
	switch d.Type {
	case "mock":
		return &Mock{make(chan bool), healthy}
	case "http":
		return makeFunChecker(
			checkHTTP(d.HTTPAddr, d.HTTPTimeout),
			d.Delay,
			d.Timeout,
			healthy,
		)
	}
	panic(fmt.Sprintf("bad type %#v in health definition", d.Type))
}

type Checker interface {
	// Check runs asynchronous health checking.  The updates channel
	// receives updates on the health state.
	Start()
	Stop() bool // returns whether it was finally healthy
	Updates() <-chan bool
}

type Mock struct {
	c chan bool
	healthy bool
}

var _ Checker = &Mock{}

func (m *Mock) Start() {}

func (m *Mock) SetHealthy(h bool) {
	if m.healthy == h {
		return
	}
	m.healthy = h
	m.c <- h
}

func (m *Mock) Stop() bool {
	close(m.c)
	return m.healthy
}

func (m *Mock) Updates() <-chan bool {
	return m.c
}

type funChecker struct {
	check   func() bool
	delay   time.Duration
	timeout time.Duration

	healthy bool
	updates chan bool
	quit    chan struct{}
}

var _ Checker = (*funChecker)(nil)

func makeFunChecker(
	check func() bool,
	delay time.Duration,
	timeout time.Duration,
	healthy bool,
) *funChecker {
	return &funChecker{
		check:   check,
		delay:   delay,
		timeout: timeout,
		healthy: healthy,
		updates: make(chan bool),
		quit:    make(chan struct{}),
	}
}

func (f *funChecker) Start() {
	defer func() {
		close(f.updates)
	}()

	start := time.Now()
	onTick := func() {
		if f.check() {
			start = time.Now()
			if !f.healthy {
				f.healthy = true
				f.updates <- true
			}
		} else if f.healthy && time.Now().After(start.Add(f.timeout)) {
			f.healthy = false
			f.updates <- false
		}
	}

	ticker := time.NewTicker(f.delay)
	defer ticker.Stop()

	onTick()
	for {
		select {
		case <-f.quit:
			return
		case <-ticker.C:
			onTick()
		}
	}
}

func (f *funChecker) Stop() bool {
	f.quit <- struct{}{}
	close(f.quit)
	return f.healthy
}

func (f *funChecker) Updates() <-chan bool {
	return f.updates
}

// Callback is a wrapper around Checker which makes callbacks to consume
// updates.
func Callback(c Checker, onUp func(), onDown func()) {
	u := c.Updates()
	go func() {
		for {
			up, ok := <-u
			if !ok {
				return
			}
			if up {
				onUp()
			} else {
				onDown()
			}
		}
	}()
}

// checkHTTP performs a health check by searching for the string "healthy" in
// the HTTP response body
func checkHTTP(addr string, timeout time.Duration) func() bool {
	return func() bool {
		client := http.Client{
			Timeout: timeout,
		}

		resp, err := client.Get(addr)
		// Check if response timeouts or returns an HTTP error
		if err != nil {
			return false
		}
		defer resp.Body.Close()
		bytes, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return false
		}

		if strings.Contains(string(bytes), "healthy") {
			return true
		}

		return false
	}
}
