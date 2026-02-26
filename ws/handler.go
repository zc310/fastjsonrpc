package ws

import (
	"fmt"
	"log/slog"
	"strings"

	"github.com/valyala/fasthttp"
)

// isJSONContentType 检查 Content-Type 是否为 application/json
func isJSONContentType(contentType string) bool {
	// 移除空格并转换为小写
	contentType = strings.ToLower(strings.TrimSpace(contentType))

	// 检查是否以 application/json 开头
	if strings.HasPrefix(contentType, "application/json") {
		return true
	}

	// 支持其他常见的 JSON Content-Type
	if strings.HasPrefix(contentType, "text/json") {
		return true
	}

	// 支持 application/json-rpc
	if strings.HasPrefix(contentType, "application/json-rpc") {
		return true
	}

	return false
}

// HTTPHandler 创建 HTTP POST JSON-RPC 处理器
func HTTPHandler(rpc *JSONRPC2) fasthttp.RequestHandler {
	return func(ctx *fasthttp.RequestCtx) {
		// 只处理 POST 请求
		if !ctx.IsPost() {
			ctx.SetStatusCode(fasthttp.StatusMethodNotAllowed)
			ctx.SetBodyString("Method Not Allowed")
			return
		}

		// 检查 Content-Type（支持 charset 参数）
		contentType := string(ctx.Request.Header.ContentType())
		if !isJSONContentType(contentType) {
			ctx.SetStatusCode(fasthttp.StatusUnsupportedMediaType)
			ctx.SetBodyString("Unsupported Media Type - expected application/json")
			return
		}

		// 获取请求体
		body := ctx.PostBody()
		if len(body) == 0 {
			ctx.SetStatusCode(fasthttp.StatusBadRequest)
			ctx.SetBodyString("Empty request body")
			return
		}

		// 设置响应头
		ctx.Response.Header.Set("Content-Type", "application/json; charset=utf-8")
		ctx.Response.Header.Set("Access-Control-Allow-Origin", "*")
		ctx.Response.Header.Set("Access-Control-Allow-Methods", "POST, OPTIONS")
		ctx.Response.Header.Set("Access-Control-Allow-Headers", "Content-Type")

		// 处理 JSON-RPC 请求
		response, err := rpc.HandleMessage(body)
		if err != nil {
			slog.Error(fmt.Sprintf("RPC handle error: %v", err))
			// 创建错误响应
			arena := rpc.arenaPool.Get()
			defer rpc.arenaPool.Put(arena)

			errorResponse, err := rpc.createErrorResponse(nil, -32603, "Internal error", err.Error())
			if err != nil {
				ctx.SetStatusCode(fasthttp.StatusInternalServerError)
				ctx.SetBodyString("Internal Server Error")
				return
			}

			ctx.SetStatusCode(fasthttp.StatusOK)
			ctx.SetBody(errorResponse)
			return
		}

		// 如果是通知（没有响应），返回空响应
		if response == nil {
			ctx.SetStatusCode(fasthttp.StatusNoContent)
			return
		}

		// 返回成功响应
		ctx.SetStatusCode(fasthttp.StatusOK)
		ctx.SetBody(response)
	}
}

// HandlerWithCORS 创建支持 CORS 的 HTTP POST JSON-RPC 处理器
func HandlerWithCORS(rpc *JSONRPC2) fasthttp.RequestHandler {
	return func(ctx *fasthttp.RequestCtx) {
		// 处理 CORS 预检请求
		if string(ctx.Method()) == "OPTIONS" {
			ctx.Response.Header.Set("Access-Control-Allow-Origin", "*")
			ctx.Response.Header.Set("Access-Control-Allow-Methods", "POST, OPTIONS")
			ctx.Response.Header.Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With")
			ctx.Response.Header.Set("Access-Control-Max-Age", "86400")
			ctx.SetStatusCode(fasthttp.StatusNoContent)
			return
		}

		// 调用普通的 HTTP 处理器
		HTTPHandler(rpc)(ctx)

		// 确保 CORS 头在普通请求中也设置
		if ctx.Response.StatusCode() == fasthttp.StatusOK {
			ctx.Response.Header.Set("Access-Control-Allow-Origin", "*")
		}
	}
}
