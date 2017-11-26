package health

import (
	"io/ioutil"
	"net/http"
	"strings"
	"time"
)

// CheckFun is a wrapper around Check using callback functions.
func CheckFun(service string, healthCheckFunc func() bool,
	onUp func(), onDown func(),
	pollDelay time.Duration,
	healthTimeout time.Duration, quit <-chan struct{}) {
	updates := make(chan bool)
	Check(healthCheckFunc, pollDelay, healthTimeout, updates, quit)
	go func() {
		for {
			up, ok := <-updates
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

// Check runs asynchronous health checking.  The updates channel
// receives updates on the health state of the backend.  Write to the
// quit channel to kill the health checker.
//
// Check assumes that the backend is unhealthy initially, and becomes
// unhealthy (if it was not already so) when it is killed.
func Check(
	healthCheckFunc func() bool,
	pollDelay time.Duration,
	healthTimeout time.Duration,
	updates chan<- bool,
	quit <-chan struct{},
) {
	go check(healthCheckFunc, pollDelay,
		healthTimeout, updates, quit)
}

func check(
	healthCheckFunc func() bool,
	pollDelay time.Duration,
	healthTimeout time.Duration,
	updates chan<- bool,
	quit <-chan struct{},
) {
	healthy := false
	defer func() {
		if healthy {
			updates <- false
		}
		close(updates)
	}()

	start := time.Now()

	onTick := func() {
		if healthCheckFunc() {
			start = time.Now()
			if !healthy {
				healthy = true
				updates <- true
			}
		} else if healthy && time.Now().After(start.Add(healthTimeout)) {
			healthy = false
			updates <- false
		}
	}

	ticker := time.NewTicker(pollDelay)
	defer ticker.Stop()

	onTick()
	for {
		select {
		case <-quit:
			return
		case <-ticker.C:
			onTick()
		}
	}
}

// HTTP performs a health check by searching for the string "healthy" in
// the HTTP response body
func HTTP(healthService string, httpTimeout time.Duration) bool {
	client := http.Client{
		Timeout: httpTimeout,
	}

	resp, err := client.Get(healthService)
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
