package health

import (
	"io/ioutil"
	"net/http"
	"strings"
	"time"
)

// CheckFun is a wrapper around Check using callback functions.
func CheckFun(service string, onUp func(), onDown func(),
	pollDelay time.Duration, timeout time.Duration,
	quit <-chan struct{}) {
	updates := make(chan bool)
	Check(service, pollDelay, timeout, updates, quit)
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
	timeout time.Duration,
	updates chan<- bool,
	quit <-chan struct{},
) {
	go check(healthService, pollDelay, timeout, updates, quit)
}

func check(
	healthService string,
	pollDelay time.Duration,
	timeout time.Duration,
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
		case t := <-ticker.C:
			if health(healthService) {
				start = t
				if !healthy {
					healthy = true
					updates <- true
				}
			}

			if healthy && t.After(start.Add(timeout)) {
				healthy = false
				updates <- false
			}
		}
	}
}

// health checks the given health service
func health(healthService string) bool {
	const timeout = time.Duration(3 * time.Second)

	client := http.Client{
		Timeout: timeout,
	}

	resp, err := client.Get(healthService)
	// Check if timeout or HTTP error
	if resp == nil && err != nil {
		return false
	}

	bytes, _ := ioutil.ReadAll(resp.Body)
	resp.Body.Close()

	if strings.Contains(string(bytes), "healthy") {
		return true
	}

	return false
}
