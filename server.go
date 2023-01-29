package fastjsonrpc

import (
	"fmt"
	"github.com/valyala/fasthttp"
	"github.com/valyala/fastjson"
)

func (p *ServerMap) Handler(ctx *fasthttp.RequestCtx) {
	ctx.Response.Header.Set("Content-Type", "application/json; charset=UTF-8")
	defer func() {
		if recover() != nil {
			_, _ = ctx.Write(ErrInternal)
		}
	}()
	p.Call(ctx)
}
func (p *ServerMap) Call(ctx *fasthttp.RequestCtx) {
	c := GetContext()
	defer func() {
		_, _ = c.w.WriteTo(ctx)
		PutContext(c)
	}()

	c.Ctx = ctx

	var err error
	if c.request, err = c.pr.ParseBytes(ctx.PostBody()); err != nil {
		_, _ = c.w.Write(ErrParse)
		return
	}

	if c.request.Type() == fastjson.TypeArray {
		var a []*fastjson.Value
		a, _ = c.request.Array()
		if len(a) > 32 || len(a) == 0 {
			_, _ = c.w.Write(ErrInvalidRequest)
			return
		}
		p.batch(a, c)
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

	f := p.getFun(string(c.Method))
	if f == nil {
		c.Error = ErrMethodNotFound
		c.writeError(c.w)
		return
	}
	f(c)

	if c.Error == nil {
		c.writeResult(c.w)
	} else {
		c.writeError(c.w)
	}
}
func (p *ServerMap) batch(a []*fastjson.Value, ctx *Context) {
	bf := GetBatchBuffer(len(a))

	for i, sc := range a {
		ct := bf.Ct[i]
		ct.Ctx = ctx.Ctx

		ct.setRequest(sc)
		if ct.request.Type() != fastjson.TypeObject || len(ct.Method) == 0 {
			_, _ = bf.B[i].Write(ErrInvalidRequest)
			continue
		}
		f := p.getFun(string(ct.Method))
		if f == nil {
			ct.Error = ErrMethodNotFound
			ct.writeError(bf.B[i])
			continue
		}

		bf.wg.Add(1)

		go func(index int) {
			cc := bf.Ct[index]

			f(cc)

			if cc.Error == nil {
				cc.writeResult(bf.B[index])
			} else {
				cc.writeError(bf.B[index])
			}

			bf.wg.Done()
		}(i)
	}
	bf.wg.Wait()

	_, _ = bf.w.WriteString("[")
	var n int
	for _, b := range bf.B {
		if b.Len() == 0 {
			continue
		}
		if n > 0 {
			_, _ = bf.w.WriteString(",")
		}
		_, _ = b.WriteTo(bf.w)
		n++
	}

	if n > 0 {
		_, _ = fmt.Fprintf(bf.w, "]")
		_, _ = bf.w.WriteTo(ctx.Ctx)
	}

	PutBatchBuffer(bf)
}
