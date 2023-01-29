package fastjsonrpc_test

import (
	"github.com/stretchr/testify/assert"
	"github.com/tidwall/pretty"
	"github.com/valyala/fasthttp"
	"github.com/valyala/fastjson"
	. "github.com/zc310/fastjsonrpc"
	"testing"
)

func TestSpec(t *testing.T) {
	t.Parallel()

	s := new(ServerMap)
	s.RegisterHandler("subtract", func(c *Context) {
		switch c.Params.Type() {
		case fastjson.TypeArray:
			c.Result = c.Params.GetInt("0") - c.Params.GetInt("1")
		case fastjson.TypeObject:
			c.Result = c.Params.GetInt("minuend") - c.Params.GetInt("subtrahend")
		}
	})
	s.RegisterHandler("sum", func(c *Context) {
		var result int
		for _, param := range c.Params.GetArray() {
			result += param.GetInt()
		}
		c.Result = result
	})
	s.RegisterHandler("get_data", func(c *Context) { c.Result = []any{"hello", 5} })
	s.RegisterHandler("notify_hello", func(c *Context) {})
	s.RegisterHandler("notify_sum", func(c *Context) {})

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
