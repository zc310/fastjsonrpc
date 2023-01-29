package fastjsonrpc

var (
	ErrParse          = []byte(`{"jsonrpc":"2.0","error":{"code":-32700,"message":"Parse error"},"id":null}`)
	ErrInvalidRequest = []byte(`{"jsonrpc":"2.0","error":{"code":-32600,"message":"Invalid Request"},"id":null}`)
	ErrInternal       = []byte(`{"jsonrpc":"2.0","error":{"code":-32603,"message":"Internal error"},"id":null}`)
	ErrMethodNotFound = NewError(-32601, "Method not found")
)

type Error struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

func (p *Error) Error() string { return p.Message }

func NewError(code int, message string) *Error { return &Error{Code: code, Message: message} }
