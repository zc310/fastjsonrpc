package fastjsonrpc

import (
	"fmt"
	"github.com/valyala/bytebufferpool"
	"github.com/valyala/fasthttp"
	"github.com/valyala/fastjson"
	"sync"
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
	aa := make([]*bytebufferpool.ByteBuffer, len(a))

	lock := new(sync.Mutex)
	wg := new(sync.WaitGroup)

	for i, sc := range a {
		ct := GetContext()
		ct.Ctx = ctx.Ctx

		ct.setRequest(sc)

		if ct.request.Type() != fastjson.TypeObject || len(ct.Method) == 0 {
			b := bytebufferpool.Get()
			_, _ = b.Write(ErrInvalidRequest)

			PutContext(ct)

			lock.Lock()
			aa[i] = b
			lock.Unlock()
			continue
		}
		f := p.getFun(string(ct.Method))
		if f == nil {
			ct.Error = ErrMethodNotFound
			b := bytebufferpool.Get()
			ct.writeError(b)

			PutContext(ct)

			lock.Lock()
			aa[i] = b
			lock.Unlock()
			continue
		}

		wg.Add(1)

		go func(cc *Context, index int) {
			f(cc)
			b := bytebufferpool.Get()
			if cc.Error == nil {
				cc.writeResult(b)
			} else {
				cc.writeError(b)
			}
			PutContext(cc)

			lock.Lock()
			aa[index] = b
			lock.Unlock()

			wg.Done()
		}(ct, i)
	}
	wg.Wait()

	buf := bytebufferpool.Get()
	var n int
	for _, b := range aa {
		if b.Len() == 0 {
			continue
		}
		if n > 0 {
			_, _ = fmt.Fprintf(buf, ",")
		}
		_, _ = b.WriteTo(buf)
		bytebufferpool.Put(b)
		n++
	}

	if n > 0 {
		_, _ = fmt.Fprintf(ctx.Ctx, "[")
		_, _ = fmt.Fprintf(buf, "]")
		_, _ = buf.WriteTo(ctx.Ctx)
	}

	bytebufferpool.Put(buf)
}
