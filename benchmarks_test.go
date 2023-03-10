package fastjsonrpc_test

import (
	"github.com/valyala/fasthttp"
	. "github.com/zc310/fastjsonrpc"
	"testing"
)

func BenchmarkEchoHandler(b *testing.B) {
	b.ReportAllocs()

	s := new(ServerMap)
	s.RegisterHandler("echo", func(c *RequestCtx) { c.Result = c.Params })

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
	s.RegisterHandler("sum", func(c *RequestCtx) {
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
func BenchmarkErrorHandler(b *testing.B) {
	b.ReportAllocs()

	s := new(ServerMap)
	s.RegisterHandler("error", func(c *RequestCtx) {
		c.Error = NewError(-32000, "Server error")
	})

	ctx := new(fasthttp.RequestCtx)
	ctx.Request.Header.SetMethod(fasthttp.MethodPost)
	ctx.Request.SetBodyString(`{"jsonrpc": "2.0", "method": "error", "id": "1"}`)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		ctx.Response.ResetBody()
		s.Handler(ctx)
	}
}
func BenchmarkBatchSumHandler(b *testing.B) {
	b.ReportAllocs()

	s := new(ServerMap)
	s.RegisterHandler("sum", func(c *RequestCtx) {
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

func BenchmarkParamsUnmarshalHandler(b *testing.B) {
	type Args struct {
		A int `json:"a,omitempty"`
		B int `json:"b,omitempty"`
	}
	b.ReportAllocs()

	s := new(ServerMap)
	s.RegisterHandler("sum", func(c *RequestCtx) {
		var a Args
		if c.Error = c.ParamsUnmarshal(&a); c.Error == nil {
			c.Result = a.A + a.B
		}
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
