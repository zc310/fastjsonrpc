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

type Handler func(ctx *RequestCtx)
type RequestCtx struct {
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

func (p *RequestCtx) ParamsUnmarshal(v any) error {
	b := bytebufferpool.Get()
	defer bytebufferpool.Put(b)

	b.B = p.Params.MarshalTo(b.B)
	return json.Unmarshal(b.B, v)
}
func (p *RequestCtx) setRequest(a *fastjson.Value) {
	p.Method = a.GetStringBytes("method")

	p.request = a
	p.Params = a.Get("params")

	if id := a.Get("id"); id != nil {
		p.id = id.MarshalTo(p.id)
	}
}

func (p *RequestCtx) writeResult(w io.Writer) {
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
		if p.Error = json.NewEncoder(b).Encode(p.Result); p.Error != nil {
			p.writeError(w)
		} else {
			writenewResult(w, p.id, b.B)
		}
		bytebufferpool.Put(b)
	}
}
func (p *RequestCtx) writeError(w io.Writer) {
	if len(p.id) == 0 {
		return
	}

	switch err := p.Error.(type) {
	case *Error:
		if err.Data == nil {
			writenewError(w, p.id, err.Code, err.Message, nil)
		} else {
			switch v := err.Data.(type) {
			case *fastjson.Value:
				b := bytebufferpool.Get()
				writenewError(w, p.id, err.Code, err.Message, v.MarshalTo(b.B))
				bytebufferpool.Put(b)
			case []byte:
				writenewError(w, p.id, err.Code, err.Message, v)
			default:
				b := bytebufferpool.Get()
				if p.Error = json.NewEncoder(b).Encode(err.Data); p.Error != nil {
					p.writeError(w)
				} else {
					writenewError(w, p.id, err.Code, err.Message, b.B)
				}
				bytebufferpool.Put(b)
			}
		}
	case error:
		writenewError(w, p.id, -32000, err.Error(), nil)
	case *fastjson.Value:
		b := bytebufferpool.Get()
		writerpcError(w, p.id, err.MarshalTo(b.B))
		bytebufferpool.Put(b)
	case []byte:
		writerpcError(w, p.id, err)
	default:
		b := bytebufferpool.Get()
		_ = json.NewEncoder(b).Encode(p.Error)
		writerpcError(w, p.id, b.B)
		bytebufferpool.Put(b)
	}

}

var (
	_pool       sync.Pool
	_poolBuffer sync.Pool
)

func getContext() *RequestCtx {
	v := _pool.Get()
	if v == nil {
		return &RequestCtx{Arena: new(fastjson.Arena), pr: new(fastjson.Parser), w: bytebufferpool.Get()}
	}
	t := v.(*RequestCtx)
	t.w = bytebufferpool.Get()
	return t
}

func putContext(p *RequestCtx) {
	p.Arena.Reset()
	bytebufferpool.Put(p.w)
	p.w = nil
	p.id = p.id[:0]
	p.Error = nil
	p.Result = nil
	p.Ctx = nil

	_pool.Put(p)
}

type batchBuffer struct {
	wg sync.WaitGroup
	B  []*bytebufferpool.ByteBuffer
	Ct []*RequestCtx
	w  *bytebufferpool.ByteBuffer
}

func getBatchBuffer(n int) *batchBuffer {
	var p *batchBuffer
	v := _poolBuffer.Get()
	if v == nil {
		p = &batchBuffer{B: make([]*bytebufferpool.ByteBuffer, 0, 32), Ct: make([]*RequestCtx, 0, 32)}
	} else {
		p = v.(*batchBuffer)
	}

	p.w = bytebufferpool.Get()
	for i := 0; i < n; i++ {
		p.B = append(p.B, bytebufferpool.Get())
		p.Ct = append(p.Ct, getContext())
	}

	return p
}

func putBatchBuffer(p *batchBuffer) {
	for i, b := range p.B {
		bytebufferpool.Put(b)
		putContext(p.Ct[i])
	}
	p.B = p.B[:0]
	p.Ct = p.Ct[:0]

	bytebufferpool.Put(p.w)

	_poolBuffer.Put(p)
}
