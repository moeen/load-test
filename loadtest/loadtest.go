package loadtest

import (
	"fmt"
	"net/http"
	"os"
	"sync"
	"time"
)

// Result is the type for storing all status codes when load testing
type Result map[int]uint

// LoadTester is the main struct containing all needed information to start the load test
type LoadTester struct {
	// client is the HTTP client used in all workers to send HTTP requests
	client *http.Client
	// request is built in NewLoadTester and used in all workers
	request *http.Request

	// requests is the number of requests that should be sent
	requests int
	// concurrency is the number multiple requests to make at time.
	concurrency int

	// mu is used when accessing result
	mu *sync.Mutex
	// result is where we store returned status code when performing load test
	result Result

	// stopCh is used to stop the running goroutines to have a graceful shutdown
	stopCh chan struct{}
}

// NewLoadTester creates a new LoadTester with given params
func NewLoadTester(url string, method string, userAgent string, timeout time.Duration, requests int, concurrency int) (*LoadTester, error) {
	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		return nil, fmt.Errorf("faield to create request: %w", err)
	}

	req.Header.Set("User-Agent", userAgent)

	client := &http.Client{Timeout: timeout}

	return &LoadTester{
		client:      client,
		request:     req,
		requests:    requests,
		concurrency: concurrency,
		mu:          &sync.Mutex{},
		result:      make(Result),
		stopCh:      make(chan struct{}, concurrency),
	}, nil
}

// Stop will send a stop signal to all running goroutines
func (lt *LoadTester) Stop() {
	// We have `lt.concurrency` goroutines running.
	for i := 0; i < lt.concurrency; i++ {
		lt.stopCh <- struct{}{}
	}
}

// Result returns the load test result
func (lt *LoadTester) Result() Result {
	defer lt.mu.Unlock()
	lt.mu.Lock()

	return lt.result
}

// Start will run `concurrency` workers
func (lt *LoadTester) Start() {
	var wg sync.WaitGroup
	wg.Add(lt.concurrency)

	for i := 0; i < lt.concurrency; i++ {
		go func() {
			defer wg.Done()

			// We have `lt.requests` and `lt.concurrency` goroutines.
			// So each goroutine should perform `lt.requests / lt.concurrency` requests.
			lt.runWorker(lt.requests / lt.concurrency)
		}()
	}

	wg.Wait()
}

// runWorker sends `n` requests and also listens for stop signal. On each request, it stores the result.
func (lt *LoadTester) runWorker(n int) {
	for i := 0; i < n; i++ {
		select {
		case <-lt.stopCh:
			return
		default:
			sc, err := lt.sendRequest()
			if err != nil {
				fmt.Fprintf(os.Stderr, "request failed: %s\n", err)
				continue
			}

			lt.storeStatusCode(sc)
		}
	}
}

// sendRequest performs a request and returns the returned status code and error if there was one.
func (lt *LoadTester) sendRequest() (int, error) {
	res, err := lt.client.Do(lt.request)
	if err != nil {
		return 0, fmt.Errorf("failed to send request: %w", err)
	}

	return res.StatusCode, nil
}

// storeStatusCode stores the given status code to the result
func (lt *LoadTester) storeStatusCode(statusCode int) {
	defer lt.mu.Unlock()

	lt.mu.Lock()
	if _, ok := lt.result[statusCode]; !ok {
		lt.result[statusCode] = 0
	}

	lt.result[statusCode]++
}
