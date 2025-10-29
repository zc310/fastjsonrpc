package fastjsonrpc_test

import (
	"errors"
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/tidwall/pretty"
	"github.com/valyala/fasthttp"
	"github.com/valyala/fastjson"
	. "github.com/zc310/fastjsonrpc"
)

func TestError(t *testing.T) {
	t.Parallel()

	s := new(ServerMap)
	s.RegisterHandler("error1", func(c *RequestCtx) {
		c.Error = NewError(-32000, "Server error")
	})
	s.RegisterHandler("error2", func(c *RequestCtx) {
		c.Error = []byte(`{"code":-32000,"message":"Server error"}`)
	})
	s.RegisterHandler("error3", func(c *RequestCtx) {
		c.Error = errors.New("server error")
	})
	s.RegisterHandler("error4", func(c *RequestCtx) {
		o := c.Arena.NewObject()
		o.Set("code", c.Arena.NewNumberInt(-32000))
		o.Set("message", c.Arena.NewString("server error"))
		c.Error = o
	})
	s.RegisterHandler("error5", func(c *RequestCtx) { c.Result = math.NaN() })
	s.RegisterHandler("error6", func(c *RequestCtx) { c.Error = &Error{Data: math.NaN()} })
	f := func(request, response string) {
		ctx := new(fasthttp.RequestCtx)
		ctx.Request.Header.SetMethod(fasthttp.MethodPost)
		ctx.Request.SetBodyString(request)

		s.Handler(ctx)

		assert.Equal(t, ctx.Response.StatusCode(), fasthttp.StatusOK)
		assert.Equal(t, string(pretty.Ugly([]byte(response))), string(pretty.Ugly(ctx.Response.Body())))
	}
	t.Run("rpc error1", func(t *testing.T) {
		f(
			`{"jsonrpc": "2.0", "method": "error1", "params": {}, "id": 1}`,
			`{"jsonrpc":"2.0","error":{"code":-32000,"message":"Server error"},"id":1}`,
		)

	})
	t.Run("rpc error2", func(t *testing.T) {
		f(
			`{"jsonrpc": "2.0", "method": "error2", "params": {}, "id": 2}`,
			`{"jsonrpc":"2.0","error":{"code":-32000,"message":"Server error"},"id":2}`,
		)
	})

	t.Run("rpc error3", func(t *testing.T) {
		f(
			`{"jsonrpc": "2.0", "method": "error3", "params": {}, "id": 3}`,
			`{"jsonrpc":"2.0","error":{"code":-32000,"message":"server error"},"id":3}`,
		)
	})
	t.Run("rpc error4", func(t *testing.T) {
		f(
			`{"jsonrpc": "2.0", "method": "error4", "params": {}, "id": 4}`,
			`{"jsonrpc":"2.0","error":{"code":-32000,"message":"server error"},"id":4}`,
		)
	})
	t.Run("rpc error5", func(t *testing.T) {
		f(
			`{"jsonrpc": "2.0", "method": "error5", "params": {}, "id": 5}`,
			`{"jsonrpc":"2.0","error":{"code":-32000,"message":"json: unsupported value: NaN"},"id":5}`,
		)
	})
	t.Run("rpc error6", func(t *testing.T) {
		f(
			`{"jsonrpc": "2.0", "method": "error6", "params": {}, "id": 6}`,
			`{"jsonrpc":"2.0","error":{"code":-32000,"message":"json: unsupported value: NaN"},"id":6}`,
		)
	})
}
func TestBatch(t *testing.T) {
	s := new(ServerMap)
	s.RegisterHandler("sum", func(c *RequestCtx) {
		c.Result = c.Params.GetInt("a") + c.Params.GetInt("b")
	})

	f := func(request, response string) {
		ctx := new(fasthttp.RequestCtx)
		ctx.Request.Header.SetMethod(fasthttp.MethodPost)
		ctx.Request.SetBodyString(request)

		s.Handler(ctx)

		assert.Equal(t, ctx.Response.StatusCode(), fasthttp.StatusOK)
		assert.Equal(t, string(pretty.Ugly([]byte(response))), string(pretty.Ugly(ctx.Response.Body())))
	}
	t.Run("rpc call Batch", func(t *testing.T) {
		f(
			`
			[
			  { "jsonrpc": "2.0", "method": "sum", "params": { "a": 3, "b": 3 }, "id": 3 },
			  { "jsonrpc": "2.0", "method": "sum", "params": { "a": 6, "b": 6 }, "id": 6 },
			  { "jsonrpc": "2.0", "method": "sum", "params": { "a": 9, "b": 9 }, "id": 9 }
			]`,
			`
			[
			  { "jsonrpc": "2.0", "result": 6, "id": 3 },
			  { "jsonrpc": "2.0", "result": 12, "id": 6 },
			  { "jsonrpc": "2.0", "result": 18, "id": 9 }
			]`,
		)
	})
}
func TestSpec(t *testing.T) {
	t.Parallel()

	s := new(ServerMap)
	s.RegisterHandler("subtract", func(c *RequestCtx) {
		switch c.Params.Type() {
		case fastjson.TypeArray:
			c.Result = c.Params.GetInt("0") - c.Params.GetInt("1")
		case fastjson.TypeObject:
			c.Result = c.Params.GetInt("minuend") - c.Params.GetInt("subtrahend")
		}
	})
	s.RegisterHandler("sum", func(c *RequestCtx) {
		var result int
		for _, param := range c.Params.GetArray() {
			result += param.GetInt()
		}
		c.Result = result
	})
	s.RegisterHandler("get_data", func(c *RequestCtx) { c.Result = []any{"hello", 5} })
	s.RegisterHandler("notify_hello", func(c *RequestCtx) {})
	s.RegisterHandler("notify_sum", func(c *RequestCtx) {})

	f := func(request, response string) {
		ctx := new(fasthttp.RequestCtx)
		ctx.Request.Header.SetMethod(fasthttp.MethodPost)
		ctx.Request.SetBodyString(request)

		s.Handler(ctx)

		assert.Equal(t, ctx.Response.StatusCode(), fasthttp.StatusOK)
		assert.Equal(t, string(pretty.Ugly([]byte(response))), string(pretty.Ugly(ctx.Response.Body())))
	}

	t.Run("rpc call with positional parameters", func(t *testing.T) {
		f(
			`{"jsonrpc": "2.0", "method": "subtract", "params": [42, 23], "id": 1}`,
			`{"jsonrpc": "2.0", "result": 19, "id": 1}`,
		)
		f(
			`{"jsonrpc": "2.0", "method": "subtract", "params": [23, 42], "id": 2}`,
			`{"jsonrpc": "2.0", "result": -19, "id": 2}`,
		)
	})

	t.Run("rpc call with named parameters", func(t *testing.T) {
		f(
			`{"jsonrpc": "2.0", "method": "subtract", "params": {"subtrahend": 23, "minuend": 42}, "id": 3}`,
			`{"jsonrpc": "2.0", "result": 19, "id": 3}`,
		)
		f(
			`{"jsonrpc": "2.0", "method": "subtract", "params": {"minuend": 42, "subtrahend": 23}, "id": 4}`,
			`{"jsonrpc": "2.0", "result": 19, "id": 4}`,
		)
	})

	t.Run("a Notification", func(t *testing.T) {
		f(
			`{"jsonrpc": "2.0", "method": "update", "params": [1,2,3,4,5]}`,
			``,
		)
		f(
			`{"jsonrpc": "2.0", "method": "foobar"}`,
			``,
		)
	})

	t.Run("rpc call of non-existent method", func(t *testing.T) {
		f(
			`{"jsonrpc": "2.0", "method": "foobar", "id": "1"}`,
			`{"jsonrpc": "2.0", "error": {"code": -32601, "message": "Method not found"}, "id": "1"}`,
		)
	})

	t.Run("rpc call with invalid JSON", func(t *testing.T) {
		f(
			`{"jsonrpc": "2.0", "method": "foobar, "params": "bar", "baz]`,
			`{"jsonrpc": "2.0", "error": {"code": -32700, "message": "Parse error"}, "id": null}`,
		)
	})

	t.Run("rpc call with invalid Request object", func(t *testing.T) {
		f(
			`{"jsonrpc": "2.0", "method": 1, "params": "bar"}`,
			`{"jsonrpc": "2.0", "error": {"code": -32600, "message": "Invalid Request"}, "id": null}`,
		)
	})

	t.Run("rpc call Batch, invalid JSON", func(t *testing.T) {
		f(
			`
			[
				{"jsonrpc": "2.0", "method": "sum", "params": [1,2,4], "id": "1"},
				{"jsonrpc": "2.0", "method"
			]`,
			`{"jsonrpc": "2.0", "error": {"code": -32700, "message": "Parse error"}, "id": null}`)
	})

	t.Run("rpc call with an empty Array", func(t *testing.T) {
		f(
			`[]`,
			`{"jsonrpc": "2.0", "error": {"code": -32600, "message": "Invalid Request"}, "id": null}`,
		)
	})

	t.Run("rpc call with an invalid Batch (but not empty)", func(t *testing.T) {
		f(
			`[1]`,
			`[{"jsonrpc": "2.0", "error": {"code": -32600, "message": "Invalid Request"}, "id": null}]`,
		)

	})
	t.Run("rpc call with invalid Batch", func(t *testing.T) {

		f(
			`[1,2,3]`,
			`
			[
				{"jsonrpc": "2.0", "error": {"code": -32600, "message": "Invalid Request"}, "id": null},
				{"jsonrpc": "2.0", "error": {"code": -32600, "message": "Invalid Request"}, "id": null},
				{"jsonrpc": "2.0", "error": {"code": -32600, "message": "Invalid Request"}, "id": null}
			]`,
		)
	})
	t.Run("rpc call Batch", func(t *testing.T) {
		f(
			`
			[
				{"jsonrpc": "2.0", "method": "sum", "params": [1,2,4], "id": "1"},
				{"jsonrpc": "2.0", "method": "notify_hello", "params": [7]},
				{"jsonrpc": "2.0", "method": "subtract", "params": [42,23], "id": "2"},
				{"foo": "boo"},
				{"jsonrpc": "2.0", "method": "foo.get", "params": {"name": "myself"}, "id": "5"},
				{"jsonrpc": "2.0", "method": "get_data", "id": "9"} 
			]`,
			`
			[
				{"jsonrpc": "2.0", "result": 7, "id": "1"},
				{"jsonrpc": "2.0", "result": 19, "id": "2"},
				{"jsonrpc": "2.0", "error": {"code": -32600, "message": "Invalid Request"}, "id": null},
				{"jsonrpc": "2.0", "error": {"code": -32601, "message": "Method not found"}, "id": "5"},
				{"jsonrpc": "2.0", "result": ["hello", 5], "id": "9"}
			]`,
		)
	})

	t.Run("rpc call Batch (all notifications)", func(t *testing.T) {
		f(
			`
			[
				{"jsonrpc": "2.0", "method": "notify_sum", "params": [1,2,4]},
				{"jsonrpc": "2.0", "method": "notify_hello", "params": [7]}
            ]`,
			``,
		)
	})
}
