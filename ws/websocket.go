package ws

import (
	"errors"
	"log/slog"
	"time"

	"sync"

	"github.com/fasthttp/websocket"
	"github.com/goccy/go-json"
	"github.com/valyala/fasthttp"
)

// Handler 创建 WebSocket JSON-RPC 处理器
func Handler(rpc *JSONRPC2, upgrader *websocket.FastHTTPUpgrader) fasthttp.RequestHandler {
	return func(ctx *fasthttp.RequestCtx) {
		err := upgrader.Upgrade(ctx, func(ws *websocket.Conn) {
			startTime := time.Now()
			defer func() {
				slog.Info("WebSocket session ended",
					"event", "session_end",
					"remote_addr", ws.RemoteAddr(),
					"total_duration", time.Since(startTime).String(),
					"disconnected_at", time.Now().Format(time.RFC3339),
				)
				_ = ws.Close()
			}()

			slog.Info("WebSocket session started",
				"event", "session_start",
				"remote_addr", ws.RemoteAddr(),
				"user_agent", string(ctx.UserAgent()),
				"connected_at", startTime.Format(time.RFC3339),
			)

			// 使用 WaitGroup 来管理所有处理 goroutine
			var wg sync.WaitGroup
			// 用于在连接关闭时通知所有 goroutine
			done := make(chan struct{})
			// 用于发送响应（保证写入顺序）
			responseChan := make(chan []byte, 100)

			// 启动响应写入器
			wg.Add(1)
			go func() {
				defer wg.Done()
				for {
					select {
					case response := <-responseChan:
						err := ws.WriteMessage(websocket.TextMessage, response)
						slog.Debug("WebSocket message sent",
							"message", json.RawMessage(response),
							"remote_addr", ws.RemoteAddr(),
							"message_type", "text",
						)
						if err != nil {
							slog.Error("WebSocket write error",
								"error", err,
								"remote_addr", ws.RemoteAddr(),
							)
							return
						}
					case <-done:
						slog.Info("WebSocket write loop stopped",
							"remote_addr", ws.RemoteAddr(),
							"reason", "done_signal_received",
						)
						return
					}
				}
			}()

			// 主循环读取消息
			for {
				_, message, err := ws.ReadMessage()
				if err != nil {
					if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
						slog.Error("WebSocket read error",
							"error", err,
							"remote_addr", ws.RemoteAddr(),
							"operation", "read_message",
						)
					}
					break
				}

				slog.Debug("WebSocket message received",
					"message", json.RawMessage(message),
					"remote_addr", ws.RemoteAddr(),
				)

				// 为每个消息启动一个 goroutine 处理
				wg.Add(1)
				go func(msg []byte) {
					defer wg.Done()

					// 处理 JSON-RPC 请求
					response, err := rpc.HandleMessage(msg)
					if err != nil {
						slog.Error("RPC handle error",
							"error", err,
							"message", json.RawMessage(msg),
							"remote_addr", ws.RemoteAddr(),
							"operation", "handle_message",
						)
						errorArena := rpc.arenaPool.Get()
						if errorResponse, err := rpc.createErrorResponse(nil, -32603, "Internal error", err.Error()); err == nil {
							select {
							case responseChan <- errorResponse:
							case <-done:
								// 连接已关闭，丢弃响应
								slog.Warn("RPC error response discarded",
									"reason", "connection_closed",
									"remote_addr", ws.RemoteAddr(),
									"error", err,
								)
							}
						}
						rpc.arenaPool.Put(errorArena)
						return
					}

					// 如果是通知，不需要响应
					if response == nil {
						return
					}

					// 发送响应到写入器
					select {
					case responseChan <- response:
					case <-done:
						// 连接已关闭，丢弃响应
					}
				}(message)
			}

			// 关闭连接，通知所有 goroutine
			close(done)
			// 等待所有处理完成
			wg.Wait()
		})

		if err != nil {
			var handshakeError websocket.HandshakeError
			if errors.As(err, &handshakeError) {
				slog.Error("WebSocket handshake error:", err)
			}

			ctx.SetStatusCode(fasthttp.StatusBadRequest)
			ctx.SetBodyString("WebSocket upgrade failed")
		}
	}
}
