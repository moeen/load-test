package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"github.com/moeen/load-test/loadtest"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"time"
)

var (
	reqFlag    = flag.Int("requests", 500, "")
	conFlag    = flag.Int("concurrency", 100, "")
	methodFlag = flag.String("method", http.MethodGet, "")
	uaFlag     = flag.String("user-agent", "moeen/load-test", "")
	toFlag     = flag.Int("time-out", 10, "")
)

const usage = `Usage: load-test [options] [http[s]://]hostname[:port]/path

Options:
	-requests Number of requests to make. Default is 500.
	-concurrency Number of multiple requests to make at a time. Default is 100.
	-method HTTP Method. Default is GET.
	-user-agent User-Agent header to send requests with. Default is moeen/load-test.
	-time-out Timeout for HTTP requests (in seconds). Use 0 for no timeouts. Default is 10 seconds. 
`

func main() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, usage)
	}

	flag.Parse()
	if flag.NArg() < 1 {
		fmt.Fprintf(os.Stderr, "missing URL\n\n")
		flag.Usage()
		os.Exit(1)
	}

	reqs := *reqFlag
	con := *conFlag

	if reqs < 1 {
		fmt.Fprintf(os.Stderr, "-requests can't be less than 1\n\n")
		flag.Usage()
		os.Exit(1)
	}

	if con < 1 {
		fmt.Fprintf(os.Stderr, "-concurrent can't be less than 1\n\n")
		flag.Usage()
		os.Exit(1)
	}

	method := strings.ToUpper(*methodFlag)
	ua := *uaFlag
	to := time.Duration(*toFlag) * time.Second

	u := flag.Args()[0]
	if _, err := url.ParseRequestURI(u); err != nil {
		fmt.Fprintf(os.Stderr, "invalid URL: %s\n\n", u)
		flag.Usage()
		os.Exit(1)
	}

	lt, err := loadtest.NewLoadTester(u, method, ua, to, reqs, con)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n\n", err)
		os.Exit(1)
	}

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)

	go func() {
		<-c
		lt.Stop()
	}()

	lt.Start()

	result, err := json.Marshal(lt.Result())
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n\n", err)
		os.Exit(1)
	}

	fmt.Fprintf(os.Stderr, "%s\n", string(result))
}
