# fast jsonrpc  [![GoDoc](https://godoc.org/github.com/zc310/fastjsonrpc?status.svg)](http://godoc.org/github.com/zc310/fastjsonrpc) [![Go Report](https://goreportcard.com/badge/github.com/zc310/fastjsonrpc)](https://goreportcard.com/report/github.com/zc310/fastjsonrpc)

## Benchmarks

```text
$ GOMAXPROCS=1 go test -bench=. -benchmem -benchtime=10s
goos: linux
goarch: amd64
pkg: github.com/zc310/fastjsonrpc
cpu: Intel(R) Core(TM) i7-4800MQ CPU @ 2.70GHz
BenchmarkEchoHandler     	21035839	       570.4 ns/op	       0 B/op	       0 allocs/op
BenchmarkSumHandler      	16349688	       724.4 ns/op	       0 B/op	       0 allocs/op
BenchmarkBatchSumHandler 	 5367964	      2219 ns/op	     712 B/op	      11 allocs/op
PASS
ok  	github.com/zc310/fastjsonrpc	39.345s

```
## Install

```
go get -u github.com/zc310/fastjsonrpc
```

## Example

```go
package main

import (
	"github.com/fasthttp/router"
	"github.com/valyala/fasthttp"
	"github.com/zc310/fastjsonrpc"
	"log"
)

func main() {
	r := router.New()
	var ss fastjsonrpc.ServerMap

	ss.RegisterHandler("echo", func(c *fastjsonrpc.Context) {
		c.Result = c.Params
	})

	ss.RegisterHandler("sum", func(c *fastjsonrpc.Context) {
		c.Result = c.Params.GetInt("a") + c.Params.GetInt("b")
	})

	r.POST("/rpc", fasthttp.CompressHandler(ss.Handler))
	log.Fatal(fasthttp.ListenAndServe(":8080", r.Handler))
}

```