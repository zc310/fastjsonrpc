package ws

import (
	"fmt"
)

// RPCError JSON-RPC 错误结构
type RPCError struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// Error 实现 error 接口
func (e *RPCError) Error() string {
	if e.Data != nil {
		return fmt.Sprintf("RPCError %d: %s (%v)", e.Code, e.Message, e.Data)
	}
	return fmt.Sprintf("RPCError %d: %s", e.Code, e.Message)
}

// 预定义的错误类型
var (
	ErrParseError     = &RPCError{Code: -32700, Message: "Parse error"}
	ErrInvalidRequest = &RPCError{Code: -32600, Message: "Invalid Request"}
	ErrMethodNotFound = &RPCError{Code: -32601, Message: "Method not found"}
	ErrInvalidParams  = &RPCError{Code: -32602, Message: "Invalid params"}
	ErrInternalError  = &RPCError{Code: -32603, Message: "Internal error"}
)

// NewRPCError 创建新的 RPC 错误
func NewRPCError(code int, message string, data interface{}) *RPCError {
	return &RPCError{
		Code:    code,
		Message: message,
		Data:    data,
	}
}

// IsRPCError 检查错误是否为 RPCError
func IsRPCError(err error) bool {
	_, ok := err.(*RPCError)
	return ok
}

// GetRPCError 获取 RPCError，如果不是则包装为 InternalError
func GetRPCError(err error) *RPCError {
	if rpcErr, ok := err.(*RPCError); ok {
		return rpcErr
	}
	return &RPCError{
		Code:    -32603,
		Message: "Internal error",
		Data:    err.Error(),
	}
}
