# fastjsonrpc

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