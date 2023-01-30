package fastjsonrpc

import "strconv"

var (
	errParse          = []byte(`{"jsonrpc":"2.0","error":{"code":-32700,"message":"Parse error"},"id":null}`)
	errInvalidRequest = []byte(`{"jsonrpc":"2.0","error":{"code":-32600,"message":"Invalid Request"},"id":null}`)
	errInternal       = []byte(`{"jsonrpc":"2.0","error":{"code":-32603,"message":"Internal error"},"id":null}`)
	errMethodNotFound = NewError(-32601, "Method not found")
)

type Error struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

func (p *Error) Error() string { return strconv.Itoa(p.Code) + ": " + p.Message }

func NewError(code int, message string) *Error { return &Error{Code: code, Message: message} }
