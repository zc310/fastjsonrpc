package main

import (
	"errors"
	"github.com/zc310/fastjsonrpc"
	"log"

	"github.com/fasthttp/router"
	"github.com/valyala/fasthttp"
)

func Index(ctx *fasthttp.RequestCtx) {
	_, _ = ctx.WriteString("Hello, world!")
}

func main() {
	r := router.New()
	r.GET("/", Index)

	var ss fastjsonrpc.ServerMap

	var tt Arith
	_ = ss.Register(&tt)
	ss.RegisterHandler("echo", func(c *fastjsonrpc.Context) {
		c.Result = c.Params
	})
	ss.RegisterHandler("sum", func(c *fastjsonrpc.Context) {
		c.Result = c.Params.GetInt("a") + c.Params.GetInt("b")
	})
	r.POST("/rpc", fasthttp.CompressHandler(ss.Handler))
	r.POST("/handler", fastjsonrpc.Rpc(handler))

	log.Fatal(fasthttp.ListenAndServe(":8080", r.Handler))
}

type Arith int

func (t *Arith) Add(c *fastjsonrpc.Context) {
	c.Result = c.Arena.NewNumberInt(c.Params.GetInt("a") + c.Params.GetInt("b"))
}

func (t *Arith) Mul(c *fastjsonrpc.Context) {
	c.Result = c.Arena.NewNumberInt(c.Params.GetInt("a") * c.Params.GetInt("b"))
}

func (t *Arith) Div(c *fastjsonrpc.Context) {
	if c.Params.GetInt("b") == 0 {
		c.Error = errors.New("divide by zero")
		return
	}
	c.Result = c.Arena.NewNumberInt(c.Params.GetInt("a") / c.Params.GetInt("b"))
}
func (t *Arith) Panic(*fastjsonrpc.Context)   { panic("ERROR") }
func (t *Arith) Error(c *fastjsonrpc.Context) { c.Error = fastjsonrpc.NewError(0, "123") }
func handler(c *fastjsonrpc.Context) {
	c.Result = c.Arena.NewNumberInt(c.Params.GetInt("a") + c.Params.GetInt("b"))
}
