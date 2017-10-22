package health

import (
	"io/ioutil"
	"net/http"
	"strings"
	"time"
)

// CheckFun is a wrapper around Check using callback functions.
func CheckFun(service string, onUp func(), onDown func(),
	pollDelay time.Duration, httpTimeout time.Duration,
	healthTimeout time.Duration, quit <-chan struct{}) {
	updates := make(chan bool)
	Check(service, pollDelay, httpTimeout, healthTimeout, updates, quit)
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
	healthService string,
	pollDelay time.Duration,
	httpTimeout time.Duration,
	healthTimeout time.Duration,
	updates chan<- bool,
	quit <-chan struct{},
) {
	go check(healthService, pollDelay, httpTimeout,
		healthTimeout, updates, quit)
}

func check(
	healthService string,
	pollDelay time.Duration,
	httpTimeout time.Duration,
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
	ticker := time.NewTicker(pollDelay)
	defer ticker.Stop()

	for {
		select {
		case <-quit:
			return
		case <-ticker.C:
			if health(healthService, httpTimeout) {
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
	}
}

// health checks the given health service
func health(healthService string, httpTimeout time.Duration) bool {
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
