# Load Test
Load test given URL concurrently and print the returned status codes.

## Build
Build the cmd:
```sh
go build -o load-test
```

## Usage
```
Usage: load-test [options] [http[s]://]hostname[:port]/path

Options:
	-requests Number of requests to make. Default is 500.
	-concurrency Number of multiple requests to make at a time. Default is 100.
	-method HTTP Method. Default is GET.
	-user-agent User-Agent header to send requests with. Default is moeen/load-test.
	-time-out Timeout for HTTP requests (in seconds). Use 0 for no timeouts. Default is 10 seconds.
```

### Example
Running 100 requests with 20 concurrent requests:
```sh
./load-test -requests 100 -concurrency 20 "https://jsonplaceholder.typicode.com/todos/1"
```

## Run Uint Tests
```sh
## Run the tests and save the coverage result
go test -coverprofile=coverage.out ./...

## Analyze the result and see the coverage
go tool cover -func=coverage.out
```