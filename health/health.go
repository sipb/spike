package health

import (
	"io/ioutil"
	"net/http"
	"strings"
	"time"
)

type Def struct {
	// valid types are "none", "http"
	Type string

	// ignored for type "none"
	Delay   time.Duration
	Timeout time.Duration

	// only for type "http"
	HTTPAddr    string
	HTTPTimeout time.Duration
}

func MakeChecker(d Def) Checker {
	switch d.Type {
	case "none":
		return Null(make(chan bool))
	case "http":
		return makeFunChecker(
			checkHTTP(d.HTTPAddr, d.HTTPTimeout),
			d.Delay,
			d.Timeout,
		)
	}
	panic("bad type in health definition")
}

type Checker interface {
	// Check runs asynchronous health checking.  The updates channel
	// receives updates on the health state.
	Start()
	Stop()
	Updates() <-chan bool
	Healthy() bool
}

type Null chan bool

var _ Checker = Null(nil)

func (n Null) Start() {
	n <- true
}

func (n Null) Stop() {
	n <- false
	close(n)
}

func (n Null) Updates() <-chan bool {
	return n
}

func (n Null) Healthy() bool {
	return true
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
) *funChecker {
	return &funChecker{
		check:   check,
		delay:   delay,
		timeout: timeout,
		healthy: false,
		updates: make(chan bool),
		quit:    make(chan struct{}),
	}
}

func (f *funChecker) Start() {
	defer func() {
		if f.healthy {
			f.updates <- false
		}
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

func (f *funChecker) Stop() {
	close(f.quit)
}

func (f *funChecker) Updates() <-chan bool {
	return f.updates
}

func (f *funChecker) Healthy() bool {
	return f.healthy
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
