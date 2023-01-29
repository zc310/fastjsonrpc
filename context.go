package fastjsonrpc

import (
	"github.com/goccy/go-json"
	"github.com/valyala/bytebufferpool"
	"github.com/valyala/fasthttp"
	"github.com/valyala/fastjson"
	"io"
	"sync"
)

//go:generate qtc -dir=.

type Handler func(ctx *Context)
type Context struct {
	request *fastjson.Value
	id      []byte
	pr      *fastjson.Parser
	w       *bytebufferpool.ByteBuffer

	Ctx   *fasthttp.RequestCtx
	Arena *fastjson.Arena

	Method []byte
	Params *fastjson.Value

	Error  any
	Result any
}

func (p *Context) setRequest(a *fastjson.Value) {
	p.Method = a.GetStringBytes("method")

	p.request = a
	p.Params = a.Get("params")

	if id := a.Get("id"); id != nil {
		p.id = id.MarshalTo(p.id)
	}
}

func (p *Context) writeResult(w io.Writer) {
	if len(p.id) == 0 {
		return
	}
	switch v := p.Result.(type) {
	case *fastjson.Value:
		b := bytebufferpool.Get()
		writenewResult(w, p.id, v.MarshalTo(b.B))
		bytebufferpool.Put(b)
	case []byte:
		writenewResult(w, p.id, v)
	default:
		b := bytebufferpool.Get()
		_ = json.NewEncoder(b).Encode(p.Result)
		writenewResult(w, p.id, b.B)
		bytebufferpool.Put(b)
	}
}
func (p *Context) writeError(w io.Writer) {
	if len(p.id) == 0 {
		return
	}

	switch err := p.Error.(type) {
	case *Error:
		if err.Data == nil {
			writenewError(w, p.id, err.Code, err.Message, nil)
			return
		}

		switch v := err.Data.(type) {
		case *fastjson.Value:
			b := bytebufferpool.Get()
			writenewError(w, p.id, err.Code, err.Message, v.MarshalTo(b.B))
			bytebufferpool.Put(b)
		case []byte:
			writenewError(w, p.id, err.Code, err.Message, v)
		default:
			b := bytebufferpool.Get()
			_ = json.NewEncoder(b).Encode(err.Data)
			writenewError(w, p.id, err.Code, err.Message, b.B)
			bytebufferpool.Put(b)
		}

	case error:
		writenewError(w, p.id, 0, err.Error(), nil)
	}

}

var (
	_pool sync.Pool
)

func GetContext() *Context {
	v := _pool.Get()
	if v == nil {
		return &Context{Arena: new(fastjson.Arena), pr: new(fastjson.Parser), w: bytebufferpool.Get()}
	}
	t := v.(*Context)
	t.w = bytebufferpool.Get()
	return t
}

func PutContext(p *Context) {
	p.Arena.Reset()
	bytebufferpool.Put(p.w)
	p.w = nil
	p.id = p.id[:0]
	p.Error = nil
	p.Result = nil

	_pool.Put(p)
}
