package fastjsonrpc_test

import (
	"github.com/valyala/fasthttp"
	. "github.com/zc310/fastjsonrpc"
	"testing"
)

func BenchmarkEchoHandler(b *testing.B) {
	b.ReportAllocs()

	s := new(ServerMap)
	s.RegisterHandler("echo", func(c *Context) { c.Result = c.Params })

	ctx := new(fasthttp.RequestCtx)
	ctx.Request.Header.SetMethod(fasthttp.MethodPost)
	ctx.Request.SetBodyString(`{"jsonrpc":"2.0","method":"echo","params":"hello","id":3}`)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ctx.Response.ResetBody()
		s.Handler(ctx)
	}
}
func BenchmarkSumHandler(b *testing.B) {
	b.ReportAllocs()

	s := new(ServerMap)
	s.RegisterHandler("sum", func(c *Context) {
		c.Result = c.Params.GetInt("a") + c.Params.GetInt("b")
	})

	ctx := new(fasthttp.RequestCtx)
	ctx.Request.Header.SetMethod(fasthttp.MethodPost)
	ctx.Request.SetBodyString(`{"jsonrpc":"2.0","method":"sum","params":{"a":3,"b":6},"id":9}`)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		ctx.Response.ResetBody()
		s.Handler(ctx)
	}
}
func BenchmarkBatchSumHandler(b *testing.B) {
	b.ReportAllocs()

	s := new(ServerMap)
	s.RegisterHandler("sum", func(c *Context) {
		c.Result = c.Params.GetInt("a") + c.Params.GetInt("b")
	})

	ctx := new(fasthttp.RequestCtx)
	ctx.Request.Header.SetMethod(fasthttp.MethodPost)
	ctx.Request.SetBodyString(`k
[
  { "jsonrpc": "2.0", "method": "sum", "params": { "a": 3, "b": 3 }, "id": 3 },
  { "jsonrpc": "2.0", "method": "sum", "params": { "a": 6, "b": 6 }, "id": 6 },
  { "jsonrpc": "2.0", "method": "sum", "params": { "a": 9, "b": 9 }, "id": 9 }
]`)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		ctx.Response.ResetBody()
		s.Handler(ctx)
	}
}
