package loadtest

import (
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestNewLoadTester(t *testing.T) {
	t.Parallel()

	var (
		url         = "https://google.com"
		method      = http.MethodGet
		userAgent   = "user-agent"
		timeout     = time.Second
		requests    = 100
		concurrency = 5
	)

	lt, err := NewLoadTester(url, method, userAgent, timeout, requests, concurrency)
	assert.NoError(t, err)

	assert.NotNil(t, lt.mu)

	assert.NotNil(t, lt.result)

	assert.NotNil(t, lt.stopCh)
	assert.Equal(t, concurrency, cap(lt.stopCh))

	assert.Equal(t, requests, lt.requests)
	assert.Equal(t, concurrency, lt.concurrency)

	req := lt.request
	assert.Equal(t, url, req.URL.String())
	assert.Equal(t, method, req.Method)
	assert.Equal(t, userAgent, req.Header.Get("User-Agent"))

	client := lt.client
	assert.Equal(t, timeout, client.Timeout)
}

func TestLoadTester_Stop(t *testing.T) {
	t.Parallel()

	var (
		url         = "https://google.com"
		method      = http.MethodGet
		userAgent   = "user-agent"
		timeout     = time.Second
		requests    = 20
		concurrency = 5
	)

	lt, err := NewLoadTester(url, method, userAgent, timeout, requests, concurrency)
	assert.NoError(t, err)

	var c uint32

	var wg sync.WaitGroup
	wg.Add(concurrency)

	for i := 0; i < concurrency; i++ {
		go func() {
			defer wg.Done()

			select {
			case <-lt.stopCh:
				atomic.AddUint32(&c, 1)
			}
		}()
	}

	lt.Stop()
	wg.Wait()

	assert.Equal(t, concurrency, int(c))
}

func TestLoadTester_Start(t *testing.T) {
	t.Parallel()

	var c uint32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddUint32(&c, 1)
	}))

	defer srv.Close()

	var (
		url         = srv.URL
		method      = http.MethodGet
		userAgent   = "user-agent"
		timeout     = time.Second
		requests    = 20
		concurrency = 5
	)

	lt, err := NewLoadTester(url, method, userAgent, timeout, requests, concurrency)
	assert.NoError(t, err)

	lt.Start()

	assert.Equal(t, requests, int(c))
}

func TestLoadTester_Result(t *testing.T) {
	t.Parallel()

	result := Result{
		200: 100,
		404: 10,
	}

	lt := &LoadTester{
		mu:     &sync.Mutex{},
		result: result,
	}

	assert.InDeltaMapValues(t, result, lt.Result(), 0)
}

func TestLoadTester_runWorker(t *testing.T) {
	t.Parallel()

	t.Run("without stopping", func(t *testing.T) {
		t.Parallel()

		var c uint32
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			atomic.AddUint32(&c, 1)
		}))

		defer srv.Close()

		var (
			url         = srv.URL
			method      = http.MethodGet
			userAgent   = "user-agent"
			timeout     = time.Second
			requests    = 20
			concurrency = 1
		)

		lt, err := NewLoadTester(url, method, userAgent, timeout, requests, concurrency)
		assert.NoError(t, err)

		lt.runWorker(requests)
		assert.Equal(t, requests, int(c))
	})

	t.Run("with stopping", func(t *testing.T) {
		t.Parallel()

		sleep := 5 * time.Millisecond

		var c uint32
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			time.Sleep(sleep)
			atomic.AddUint32(&c, 1)
		}))

		defer srv.Close()

		var (
			url         = srv.URL
			method      = http.MethodGet
			userAgent   = "user-agent"
			timeout     = time.Second
			requests    = 20
			concurrency = 1
			stopAfter   = 2
		)

		lt, err := NewLoadTester(url, method, userAgent, timeout, requests, concurrency)
		assert.NoError(t, err)

		var wg sync.WaitGroup
		wg.Add(1)

		timer := time.NewTimer(time.Duration(stopAfter) * sleep)
		go func() {
			for {
				select {
				case <-timer.C:
					lt.Stop()
					wg.Done()
				}
			}
		}()

		lt.runWorker(requests)

		wg.Wait()

		assert.Equal(t, stopAfter, int(c))
	})
}

func TestLoadTester_sendRequest(t *testing.T) {
	t.Parallel()

	statusCode := http.StatusOK

	var c uint32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(statusCode)
		atomic.AddUint32(&c, 1)
	}))

	defer srv.Close()

	var (
		url         = srv.URL
		method      = http.MethodGet
		userAgent   = "user-agent"
		timeout     = time.Second
		requests    = 20
		concurrency = 1
	)

	lt, err := NewLoadTester(url, method, userAgent, timeout, requests, concurrency)
	assert.NoError(t, err)

	sc, err := lt.sendRequest()
	assert.NoError(t, err)
	assert.Equal(t, statusCode, sc)

	assert.Equal(t, 1, int(c))
}

func TestLoadTester_storeStatusCode(t *testing.T) {
	t.Parallel()

	var (
		url         = "https://google.com"
		method      = http.MethodGet
		userAgent   = "user-agent"
		timeout     = time.Second
		requests    = 20
		concurrency = 1
		statusCodes = map[int]uint{
			200: 543,
			201: 11,
			400: 99,
			500: 10,
		}
	)

	lt, err := NewLoadTester(url, method, userAgent, timeout, requests, concurrency)
	assert.NoError(t, err)

	var wg sync.WaitGroup

	for k, v := range statusCodes {
		wg.Add(1)

		go func(c uint, sc int) {
			defer wg.Done()

			for i := 0; i < int(c); i++ {
				lt.storeStatusCode(sc)
			}
		}(v, k)
	}

	wg.Wait()

	assert.InDeltaMapValues(t, statusCodes, lt.result, 0)
}
