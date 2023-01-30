package fastjsonrpc

import (
	"github.com/valyala/fasthttp"
	"github.com/valyala/fastjson"
)

func Rpc(h Handler) fasthttp.RequestHandler {
	return func(ctx *fasthttp.RequestCtx) {
		c := getContext()
		defer func() {
			_, _ = c.w.WriteTo(ctx)
			putContext(c)
		}()

		c.Ctx = ctx
		ctx.Response.Header.Set("Content-Type", "application/json; charset=UTF-8")

		var err error
		if c.request, err = c.pr.ParseBytes(ctx.Request.Body()); err != nil {
			_, _ = c.w.Write(ErrParse)
			return
		}

		if c.request.Type() != fastjson.TypeObject {
			_, _ = c.w.Write(ErrInvalidRequest)
			return
		}
		c.setRequest(c.request)
		if len(c.Method) == 0 {
			_, _ = c.w.Write(ErrInvalidRequest)
			return
		}

		h(c)

		if c.Error == nil {
			c.writeResult(c.w)
		} else {
			c.writeError(c.w)
		}
	}
}
