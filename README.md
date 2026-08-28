# fast jsonrpc  [![GoDoc](https://godoc.org/github.com/zc310/fastjsonrpc?status.svg)](http://godoc.org/github.com/zc310/fastjsonrpc) [![Go Report](https://goreportcard.com/badge/github.com/zc310/fastjsonrpc)](https://goreportcard.com/report/github.com/zc310/fastjsonrpc)

Fast [JSON-RPC 2.0](https://www.jsonrpc.org/specification) Server based
on [fasthttp](https://github.com/valyala/fasthttp)

## Benchmarks

```text
$ GOMAXPROCS=1 go test -bench=. -benchmem -benchtime=10s
goos: linux
goarch: amd64
pkg: github.com/zc310/fastjsonrpc
cpu: Intel(R) Core(TM) i7-4800MQ CPU @ 2.70GHz
BenchmarkEchoHandler            	21422912	       555.2 ns/op	       0 B/op	       0 allocs/op
BenchmarkSumHandler             	16634361	       718.4 ns/op	       0 B/op	       0 allocs/op
BenchmarkErrorHandler           	15317910	       744.7 ns/op	      48 B/op	       1 allocs/op
BenchmarkBatchSumHandler        	 5419057	      2213 ns/op	     712 B/op	      11 allocs/op
BenchmarkParamsUnmarshalHandler 	12894405	       923.6 ns/op	      32 B/op	       2 allocs/op
PASS
ok  	github.com/zc310/fastjsonrpc	64.434s
```

## Install

```
go get -u github.com/zc310/fastjsonrpc
```

## Example

```go
package main

import (
	"errors"
	"log"
	"log/slog"
	"runtime"

	"github.com/fasthttp/router"
	"github.com/fasthttp/websocket"
	"github.com/valyala/fasthttp"
	"github.com/valyala/fastjson"
	"github.com/zc310/fastjsonrpc"
	"github.com/zc310/fastjsonrpc/ws"
)

func main() {
	r := router.New()
	r.GET("/", func(ctx *fasthttp.RequestCtx) {
		_, _ = ctx.WriteString("Hello, world!")
	})

	var ss fastjsonrpc.ServerMap

	var tt Arith
	_ = ss.Register(&tt)
	ss.RegisterHandler("echo", func(c *fastjsonrpc.RequestCtx) {
		c.Result = c.Params
	})
	ss.RegisterHandler("sum", func(c *fastjsonrpc.RequestCtx) {
		c.Result = c.Params.GetInt("a") + c.Params.GetInt("b")
	})
	ss.RegisterHandler("error", func(c *fastjsonrpc.RequestCtx) {
		c.Error = nil
	})
	r.POST("/rpc", fasthttp.CompressHandler(ss.Handler))

	r.GET("/mem", fastjsonrpc.Get(func(ctx *fastjsonrpc.RequestCtx) {
		var ms runtime.MemStats
		runtime.ReadMemStats(&ms)
		ctx.Result = ms
	}))

	rpc := ws.NewJSONRPC2()
	r.ANY("/ws", WebSocketHandler(rpc))

	rpc.RegisterMethodFunc("mem", func(params *fastjson.Value) (any, error) {
		var ms runtime.MemStats
		runtime.ReadMemStats(&ms)
		return ms, nil
	})

	_ = fasthttp.ListenAndServe(":8080", r.Handler)
}

type Arith int
type Args struct {
	A int `json:"a,omitempty"`
	B int `json:"b,omitempty"`
}

func (t *Arith) Add(c *fastjsonrpc.RequestCtx) {
	var a Args
	if c.Error = c.ParamsUnmarshal(&a); c.Error == nil {
		c.Result = a.A + a.B
	}
}

func (t *Arith) Mul(c *fastjsonrpc.RequestCtx) {
	c.Result = c.Arena.NewNumberInt(c.Params.GetInt("a") * c.Params.GetInt("b"))
}

func (t *Arith) Div(c *fastjsonrpc.RequestCtx) {
	if c.Params.GetInt("b") == 0 {
		c.Error = errors.New("divide by zero")
		return
	}
	c.Result = c.Arena.NewNumberInt(c.Params.GetInt("a") / c.Params.GetInt("b"))
}
func (t *Arith) Panic(*fastjsonrpc.RequestCtx) { panic("ERROR") }
func (t *Arith) Error(c *fastjsonrpc.RequestCtx) {
	c.Error = fastjsonrpc.NewError(-32000, "Server error")
}

func WebSocketHandler(rpc *ws.JSONRPC2) fasthttp.RequestHandler {
	rpc.RegisterTestService()

	methods := rpc.GetRegisteredMethods()
	slog.Info("Registered test methods:")
	for _, method := range methods {
		log.Printf("  - %s", method)
	}
	var upgrader = websocket.FastHTTPUpgrader{
		CheckOrigin: func(ctx *fasthttp.RequestCtx) bool {
			return true
		},
	}

	return ws.Handler(rpc, &upgrader)
}

```

### HTTP Request

```http request
### echo

POST http://localhost:8080/rpc
Content-Type: application/json

{"jsonrpc":"2.0","method":"echo","params":{"a":9,"b":9},"id":9}

### sum

POST http://localhost:8080/rpc
Content-Type: application/json

{"jsonrpc":"2.0","method":"sum","params":{"a":9,"b":9},"id":9}

### Arith.Add

POST http://localhost:8080/rpc
Content-Type: application/json

{"jsonrpc":"2.0","method":"Arith.Add","params":{"a":9,"b":9},"id":9}

### Arith.Mul

POST http://localhost:8080/rpc
Content-Type: application/json

{"jsonrpc":"2.0","method":"Arith.Mul","params":{"a":9,"b":9},"id":9}

### Arith.Div

POST http://localhost:8080/rpc
Content-Type: application/json

{"jsonrpc":"2.0","method":"Arith.Div","params":{"a":9,"b":9},"id":9}

### Arith.Error

POST http://localhost:8080/rpc
Content-Type: application/json

{"jsonrpc":"2.0","method":"Arith.Error","params":{"a":9,"b":9},"id":9}

### Arith.Panic

POST http://localhost:8080/rpc
Content-Type: application/json

{"jsonrpc":"2.0","method":"Arith.Panic","params":{"a":9,"b":9},"id":9}


### mem
GET http://localhost:8080/mem

```

### WebSocket Request

```js
<script>
  const ws = new WebSocket('ws://127.0.0.1:8080/ws');

  ws.onopen = () => {
    console.log('WebSocket 连接成功');

    const message = {
      jsonrpc: "2.0",
      method: "echo",
      params: { a: 9, b: 9 },
      id: 9
    };

    ws.send(JSON.stringify(message));
    console.log('已发送消息:', message);
  };

  ws.onmessage = (event) => {
    console.log('收到消息:', event.data);
  };

  ws.onerror = (error) => {
    console.error('WebSocket 错误:', error);
  };

  ws.onclose = () => {
    console.log('WebSocket 连接关闭');
  };
</script>
```
