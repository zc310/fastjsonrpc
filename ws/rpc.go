package ws

import (
	"errors"
	"fmt"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"sync"
	"unicode"
	"unicode/utf8"

	"github.com/goccy/go-json"
	"github.com/iancoleman/strcase"
	"github.com/valyala/bytebufferpool"
	"github.com/valyala/fastjson"
)

// RPCMethod RPC 方法类型
type RPCMethod func(arena *fastjson.Arena, params *fastjson.Value) (interface{}, error)

// JSONRPC2 JSON-RPC 2.0 处理器
type JSONRPC2 struct {
	methods    map[string]RPCMethod
	mu         sync.RWMutex
	parserPool fastjson.ParserPool
	arenaPool  fastjson.ArenaPool
}

// NewJSONRPC2 创建新的 JSON-RPC 2.0 实例
func NewJSONRPC2() *JSONRPC2 {
	return &JSONRPC2{
		methods: make(map[string]RPCMethod),
	}
}

// RegisterMethod 注册 RPC 方法
func (j *JSONRPC2) RegisterMethod(name string, method RPCMethod) {
	j.mu.Lock()
	defer j.mu.Unlock()
	j.methods[name] = method
}

// RegisterMethodFunc 注册 RPC 方法（函数适配器）
func (j *JSONRPC2) RegisterMethodFunc(name string, method func(params *fastjson.Value) (interface{}, error)) {
	j.mu.Lock()
	defer j.mu.Unlock()
	j.methods[name] = func(arena *fastjson.Arena, params *fastjson.Value) (interface{}, error) {
		return method(params)
	}
}

// HandleMessage 处理 JSON-RPC 消息
func (j *JSONRPC2) HandleMessage(message []byte) ([]byte, error) {
	// 从池中获取解析器和 arena
	parser := j.parserPool.Get()
	defer j.parserPool.Put(parser)

	arena := j.arenaPool.Get()
	defer j.arenaPool.Put(arena)

	value, err := parser.ParseBytes(message)
	if err != nil {
		return j.createParseError()
	}

	return j.handleParsedValue(arena, value)
}

// handleParsedValue 处理已解析的 JSON 值
func (j *JSONRPC2) handleParsedValue(arena *fastjson.Arena, value *fastjson.Value) ([]byte, error) {
	switch value.Type() {
	case fastjson.TypeArray:
		return j.handleBatchRequest(value)
	case fastjson.TypeObject:
		return j.handleSingleRequest(arena, value)
	default:
		return j.createErrorResponse(nil, -32600, "Invalid Request", "Request must be object or array")
	}
}

// handleBatchRequest 处理批量请求
func (j *JSONRPC2) handleBatchRequest(batchValue *fastjson.Value) ([]byte, error) {
	array, err := batchValue.Array()
	if err != nil {
		return j.createInvalidRequestError()
	}

	if len(array) == 0 {
		return j.createInvalidRequestError()
	}

	// 为每个请求创建新的 arena 来处理响应
	responses := make([][]byte, 0, len(array))

	for _, item := range array {
		if item.Type() != fastjson.TypeObject {
			// 创建错误响应
			errorResponse, _ := j.createInvalidRequestError()
			responses = append(responses, errorResponse)
			continue
		}

		itemArena := j.arenaPool.Get()
		response, err := j.handleSingleRequest(itemArena, item)
		j.arenaPool.Put(itemArena)

		if err != nil {
			// 创建内部错误响应
			errorResponse, _ := j.createInternalError(nil, err.Error())
			responses = append(responses, errorResponse)
		} else if response != nil {
			responses = append(responses, response)
		}
		// 如果是通知（response == nil），则不添加到结果中
	}

	if len(responses) == 0 {
		return nil, nil
	}

	// 手动构建批量响应数组
	batchResponse := make([]byte, 0, 1024)
	batchResponse = append(batchResponse, '[')
	for i, response := range responses {
		if i > 0 {
			batchResponse = append(batchResponse, ',')
		}
		batchResponse = append(batchResponse, response...)
	}
	batchResponse = append(batchResponse, ']')

	return batchResponse, nil
}

// createInternalError 创建内部错误响应
func (j *JSONRPC2) createInternalError(id *fastjson.Value, data interface{}) ([]byte, error) {
	buf := bytebufferpool.Get()
	defer bytebufferpool.Put(buf)

	_, _ = buf.WriteString(`{"jsonrpc":"2.0","id":`)

	if id == nil {
		_, _ = buf.WriteString("null")
	} else {
		_, _ = buf.Write(id.MarshalTo(nil))
	}

	_, _ = buf.WriteString(`,"error":{"code":-32603,"message":"Internal error"`)

	if data != nil {
		dataBytes, err := j.marshalData(data)
		if err == nil && len(dataBytes) > 0 {
			_, _ = buf.WriteString(`,"data":`)
			_, _ = buf.Write(dataBytes)
		}
	}

	_, _ = buf.WriteString("}}")

	response := make([]byte, buf.Len())
	copy(response, buf.Bytes())
	return response, nil
}

// handleSingleRequest 处理单个请求
func (j *JSONRPC2) handleSingleRequest(arena *fastjson.Arena, value *fastjson.Value) ([]byte, error) {
	// 验证 JSON-RPC 版本
	jsonrpcVal := value.Get("jsonrpc")
	if jsonrpcVal == nil || jsonrpcVal.Type() != fastjson.TypeString {
		return j.createInvalidRequestError()
	}

	if string(jsonrpcVal.GetStringBytes()) != "2.0" {
		return j.createInvalidRequestError()
	}

	// 获取方法名
	methodVal := value.Get("method")
	if methodVal == nil || methodVal.Type() != fastjson.TypeString {
		return j.createInvalidRequestError()
	}
	methodName := string(methodVal.GetStringBytes())

	// 获取 ID
	var id *fastjson.Value
	if value.Exists("id") {
		id = value.Get("id")
	}

	// 处理通知（没有 ID 的请求）
	if id == nil {
		j.handleNotification(methodName, value.Get("params"))
		return nil, nil
	}

	// 查找方法
	j.mu.RLock()
	method, exists := j.methods[methodName]
	j.mu.RUnlock()

	if !exists {
		return j.createMethodNotFoundError(id)
	}

	// 调用方法
	result, err := method(arena, value.Get("params"))
	if err != nil {
		var rpcErr *RPCError
		if errors.As(err, &rpcErr) {
			return j.createErrorResponse(id, rpcErr.Code, rpcErr.Message, rpcErr.Data)
		}
		return j.createErrorResponse(id, -32000, err.Error(), nil)
	}

	// 创建成功响应
	return j.createSuccessResponse(id, result)
}

// handleNotification 处理通知
func (j *JSONRPC2) handleNotification(methodName string, params *fastjson.Value) {
	j.mu.RLock()
	method, exists := j.methods[methodName]
	j.mu.RUnlock()

	if exists {
		go func() {
			// 为异步处理创建新的 arena
			asyncArena := j.arenaPool.Get()
			defer j.arenaPool.Put(asyncArena)

			_, _ = method(asyncArena, params)
		}()
	}
}

// createSuccessResponse 创建成功响应
func (j *JSONRPC2) createSuccessResponse(id *fastjson.Value, result interface{}) ([]byte, error) {
	buf := bytebufferpool.Get()
	defer bytebufferpool.Put(buf)

	_, _ = buf.WriteString(`{"jsonrpc":"2.0","id":`)

	// 写入 ID
	if id == nil {
		_, _ = buf.WriteString("null")
	} else {
		_, _ = buf.Write(id.MarshalTo(nil))
	}

	_, _ = buf.WriteString(`,"result":`)

	// 写入结果
	resultBytes, err := j.marshalResult(result)
	if err != nil {
		return nil, err
	}
	_, _ = buf.Write(resultBytes)

	_ = buf.WriteByte('}')

	// 返回副本，因为 buf 会被放回池中
	response := make([]byte, buf.Len())
	copy(response, buf.Bytes())
	return response, nil
}

// createErrorResponse 创建错误响应
func (j *JSONRPC2) createErrorResponse(id *fastjson.Value, code int, message string, data interface{}) ([]byte, error) {
	buf := bytebufferpool.Get()
	defer bytebufferpool.Put(buf)

	_, _ = buf.WriteString(`{"jsonrpc":"2.0","id":`)

	// 写入 ID
	if id == nil {
		_, _ = buf.WriteString("null")
	} else {
		idBytes := id.MarshalTo(nil)
		_, _ = buf.Write(idBytes)
	}

	_, _ = buf.WriteString(`,"error":{"code":`)
	_, _ = buf.WriteString(strconv.Itoa(code))
	_, _ = buf.WriteString(`,"message":`)
	jsonMessage, _ := json.Marshal(message)
	_, _ = buf.Write(jsonMessage)

	// 写入错误数据（如果有）
	if data != nil {
		dataBytes, err := j.marshalData(data)
		if err == nil && len(dataBytes) > 0 {
			_, _ = buf.WriteString(`,"data":`)
			_, _ = buf.Write(dataBytes)
		}
	}

	_, _ = buf.WriteString("}}")

	response := make([]byte, buf.Len())
	copy(response, buf.Bytes())
	return response, nil
}

// marshalID 序列化 ID
func (j *JSONRPC2) marshalID(id *fastjson.Value) ([]byte, error) {
	if id == nil {
		return []byte("null"), nil
	}
	return id.MarshalTo(nil), nil
}

// marshalResult 序列化结果
func (j *JSONRPC2) marshalResult(result interface{}) ([]byte, error) {
	if result == nil {
		return []byte("null"), nil
	}

	// 如果已经是字节数组，直接返回
	if b, ok := result.([]byte); ok {
		return b, nil
	}

	// 如果是 fastjson.Value，直接序列化
	if v, ok := result.(*fastjson.Value); ok {
		return v.MarshalTo(nil), nil
	}

	// 使用 encoding/json 处理复杂类型
	return json.Marshal(result)
}

// marshalData 序列化错误数据
func (j *JSONRPC2) marshalData(data interface{}) ([]byte, error) {
	if data == nil {
		return nil, nil
	}

	// 如果已经是字节数组，直接返回
	if b, ok := data.([]byte); ok {
		return b, nil
	}

	// 使用 encoding/json 处理复杂类型
	return json.Marshal(data)
}

// escapeJSONString 转义 JSON 字符串
func escapeJSONString(s string) string {
	b, _ := json.Marshal(s)
	// 移除外层的引号
	if len(b) >= 2 && b[0] == '"' && b[len(b)-1] == '"' {
		return string(b[1 : len(b)-1])
	}
	return string(b)
}

// createParseError 创建解析错误响应
func (j *JSONRPC2) createParseError() ([]byte, error) {
	return []byte(`{"jsonrpc":"2.0","id":null,"error":{"code":-32700,"message":"Parse error"}}`), nil
}

// createInvalidRequestError 创建无效请求错误响应
func (j *JSONRPC2) createInvalidRequestError() ([]byte, error) {
	return []byte(`{"jsonrpc":"2.0","id":null,"error":{"code":-32600,"message":"Invalid Request"}}`), nil
}

// createMethodNotFoundError 创建方法未找到错误响应
func (j *JSONRPC2) createMethodNotFoundError(id *fastjson.Value) ([]byte, error) {
	buf := bytebufferpool.Get()
	defer bytebufferpool.Put(buf)

	_, _ = buf.WriteString(`{"jsonrpc":"2.0","id":`)

	if id == nil {
		_, _ = buf.WriteString("null")
	} else {
		idBytes := id.MarshalTo(nil)

		_, _ = buf.Write(idBytes)
	}

	_, _ = buf.WriteString(`,"error":{"code":-32601,"message":"Method not found"}}`)

	response := make([]byte, buf.Len())
	copy(response, buf.Bytes())
	return response, nil
}

// RegisterObject 注册对象的所有公开方法（仅支持新签名）
func (j *JSONRPC2) RegisterObject(obj interface{}, prefix ...string) error {
	j.mu.Lock()
	defer j.mu.Unlock()

	objType := reflect.TypeOf(obj)
	objValue := reflect.ValueOf(obj)

	// 确定前缀
	var methodPrefix string
	if len(prefix) > 0 && prefix[0] != "" {
		methodPrefix = prefix[0]
	} else {
		// 使用小写类名作为前缀
		typeName := objType.String()
		if idx := strings.LastIndex(typeName, "."); idx != -1 {
			typeName = typeName[idx+1:]
		}
		methodPrefix = strcase.ToSnake(typeName) + "."
	}

	// 遍历所有方法
	for i := 0; i < objType.NumMethod(); i++ {
		method := objType.Method(i)

		// 检查方法是否可导出（首字母大写）
		if !isExported(method.Name) {
			continue
		}

		// 跳过不需要的方法
		if method.Name == "RegisterMethod" || method.Name == "RegisterObject" {
			continue
		}

		// 验证方法签名（新签名）
		if err := validateNewMethodSignature(method.Type); err != nil {
			// 跳过不符合新签名的方法
			continue
		}

		// 构建完整方法名
		methodName := methodPrefix + strcase.ToSnake(method.Name)

		// 创建方法包装器
		wrapper := j.createNewMethodWrapper(objValue, method)

		// 注册方法
		j.methods[methodName] = wrapper
	}

	return nil
}

// RegisterObjectMethods 注册对象的指定方法（仅支持新签名）
func (j *JSONRPC2) RegisterObjectMethods(obj interface{}, methodNames []string, prefix ...string) error {
	j.mu.Lock()
	defer j.mu.Unlock()

	objType := reflect.TypeOf(obj)
	objValue := reflect.ValueOf(obj)

	// 确定前缀
	var methodPrefix string
	if len(prefix) > 0 && prefix[0] != "" {
		methodPrefix = prefix[0]
	} else {
		typeName := objType.String()
		if idx := strings.LastIndex(typeName, "."); idx != -1 {
			typeName = typeName[idx+1:]
		}
		methodPrefix = strcase.ToSnake(typeName) + "."
	}

	for _, methodName := range methodNames {
		method, exists := objType.MethodByName(methodName)
		if !exists {
			return fmt.Errorf("method %s not found", methodName)
		}

		// 验证方法签名（新签名）
		if err := validateNewMethodSignature(method.Type); err != nil {
			return fmt.Errorf("method %s has invalid signature (expected func(arena *fastjson.Arena, params *fastjson.Value) (interface{}, error)): %v", methodName, err)
		}

		// 构建完整方法名
		fullMethodName := methodPrefix + strcase.ToSnake(methodName)

		// 创建方法包装器
		wrapper := j.createNewMethodWrapper(objValue, method)

		// 注册方法
		j.methods[fullMethodName] = wrapper
	}

	return nil
}

// createNewMethodWrapper 创建新签名方法包装器
func (j *JSONRPC2) createNewMethodWrapper(objValue reflect.Value, method reflect.Method) RPCMethod {
	return func(arena *fastjson.Arena, params *fastjson.Value) (interface{}, error) {
		// 调用对象方法（新签名）
		args := []reflect.Value{objValue, reflect.ValueOf(arena), reflect.ValueOf(params)}
		results := method.Func.Call(args)

		// 处理返回值
		if len(results) == 0 {
			return nil, nil
		}

		if len(results) > 1 && !results[1].IsNil() {
			return results[0].Interface(), results[1].Interface().(error)
		}

		return results[0].Interface(), nil
	}
}

// validateNewMethodSignature 验证新方法签名
func validateNewMethodSignature(methodType reflect.Type) error {
	// 检查参数数量（接收器 + arena + params）
	if methodType.NumIn() != 3 {
		return fmt.Errorf("expected 3 parameters (receiver + arena + params), got %d", methodType.NumIn())
	}

	// 检查第二个参数类型是否为 *fastjson.Arena
	arenaType := methodType.In(1)
	if arenaType != reflect.TypeOf((*fastjson.Arena)(nil)) {
		return fmt.Errorf("second parameter must be *fastjson.Arena, got %v", arenaType)
	}

	// 检查第三个参数类型是否为 *fastjson.Value
	paramType := methodType.In(2)
	if paramType != reflect.TypeOf((*fastjson.Value)(nil)) {
		return fmt.Errorf("third parameter must be *fastjson.Value, got %v", paramType)
	}

	// 检查返回值数量
	if methodType.NumOut() != 2 {
		return fmt.Errorf("expected 2 return values (result, error), got %d", methodType.NumOut())
	}

	// 检查第二个返回值类型是否为 error
	errorType := methodType.Out(1)
	if errorType != reflect.TypeOf((*error)(nil)).Elem() {
		return fmt.Errorf("second return value must be error, got %v", errorType)
	}

	return nil
}

// isExported 检查方法名是否可导出（首字母大写）
func isExported(name string) bool {
	r, _ := utf8.DecodeRuneInString(name)
	return unicode.IsUpper(r)
}

// GetRegisteredMethods 获取所有已注册的方法名（用于调试）
func (j *JSONRPC2) GetRegisteredMethods() []string {
	j.mu.RLock()
	defer j.mu.RUnlock()

	methods := make([]string, 0, len(j.methods))
	for name := range j.methods {
		methods = append(methods, name)
	}
	sort.Strings(methods)
	return methods
}
