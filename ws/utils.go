package ws

import (
	"fmt"
	"log/slog"

	"github.com/goccy/go-json"
	"github.com/valyala/fastjson"
)

// TestService 测试服务
type TestService struct {
	userCount int
}

// NewTestService 创建测试服务实例
func NewTestService() *TestService {
	return &TestService{
		userCount: 0,
	}
}

// Ping 健康检查
func (t *TestService) Ping(arena *fastjson.Arena, params *fastjson.Value) (interface{}, error) {
	return arena.NewString("pong"), nil
}

// Echo 回显参数
func (t *TestService) Echo(arena *fastjson.Arena, params *fastjson.Value) (interface{}, error) {
	if params == nil {
		return arena.NewNull(), nil
	}

	var out interface{}
	err := json.Unmarshal(params.MarshalTo(nil), &out)
	return out, err
}

// Add 加法运算
func (t *TestService) Add(arena *fastjson.Arena, params *fastjson.Value) (interface{}, error) {
	if params == nil || params.Type() != fastjson.TypeArray {
		return nil, fmt.Errorf("params must be array")
	}
	array, _ := params.Array()
	if len(array) != 2 {
		return nil, fmt.Errorf("expected 2 parameters")
	}
	a, err := array[0].Int()
	if err != nil {
		return nil, fmt.Errorf("first parameter must be integer")
	}
	b, err := array[1].Int()
	if err != nil {
		return nil, fmt.Errorf("second parameter must be integer")
	}
	return arena.NewNumberInt(a + b), nil
}

// Multiply 乘法运算
func (t *TestService) Multiply(arena *fastjson.Arena, params *fastjson.Value) (interface{}, error) {
	if params == nil || params.Type() != fastjson.TypeArray {
		return nil, fmt.Errorf("params must be array")
	}
	array, _ := params.Array()
	if len(array) != 2 {
		return nil, fmt.Errorf("expected 2 parameters")
	}
	a, err := array[0].Int()
	if err != nil {
		return nil, fmt.Errorf("first parameter must be integer")
	}
	b, err := array[1].Int()
	if err != nil {
		return nil, fmt.Errorf("second parameter must be integer")
	}
	return arena.NewNumberInt(a * b), nil
}

// Divide 除法运算
func (t *TestService) Divide(arena *fastjson.Arena, params *fastjson.Value) (interface{}, error) {
	if params == nil {
		return nil, fmt.Errorf("params is required")
	}

	var dividend, divisor float64
	var err error

	if params.Type() == fastjson.TypeObject {
		// 命名参数
		dividendVal := params.Get("dividend")
		divisorVal := params.Get("divisor")

		if dividendVal == nil || divisorVal == nil {
			return nil, fmt.Errorf("both 'dividend' and 'divisor' parameters are required")
		}

		dividend, err = dividendVal.Float64()
		if err != nil {
			return nil, fmt.Errorf("dividend must be a number")
		}

		divisor, err = divisorVal.Float64()
		if err != nil {
			return nil, fmt.Errorf("divisor must be a number")
		}
	} else if params.Type() == fastjson.TypeArray {
		// 位置参数
		array, err := params.Array()
		if err != nil {
			return nil, fmt.Errorf("invalid parameters array")
		}

		if len(array) != 2 {
			return nil, fmt.Errorf("expected 2 parameters")
		}

		dividend, err = array[0].Float64()
		if err != nil {
			return nil, fmt.Errorf("first parameter must be a number")
		}

		divisor, err = array[1].Float64()
		if err != nil {
			return nil, fmt.Errorf("second parameter must be a number")
		}
	} else {
		return nil, fmt.Errorf("params must be object or array")
	}

	if divisor == 0 {
		return nil, fmt.Errorf("division by zero")
	}

	return arena.NewNumberFloat64(dividend / divisor), nil
}

// Count 计算参数个数
func (t *TestService) Count(arena *fastjson.Arena, params *fastjson.Value) (interface{}, error) {
	if params == nil {
		return arena.NewNumberInt(0), nil
	}

	if params.Type() != fastjson.TypeArray {
		return nil, fmt.Errorf("params must be array")
	}

	array, err := params.Array()
	if err != nil {
		return nil, fmt.Errorf("invalid parameters array")
	}

	return arena.NewNumberInt(len(array)), nil
}

// SlowOperation 模拟慢操作
func (t *TestService) SlowOperation(arena *fastjson.Arena, params *fastjson.Value) (interface{}, error) {
	if params == nil || params.Type() != fastjson.TypeArray {
		return nil, fmt.Errorf("params must be array")
	}
	array, _ := params.Array()
	if len(array) != 1 {
		return nil, fmt.Errorf("expected 1 parameter")
	}
	delay, err := array[0].Int()
	if err != nil || delay < 0 {
		return nil, fmt.Errorf("delay must be non-negative integer")
	}

	if delay > 10 {
		delay = 10 // 限制最大延迟
	}

	// 在实际应用中这里会有 time.Sleep，但为了测试我们直接返回
	resultObj := arena.NewObject()
	resultObj.Set("status", arena.NewString("completed"))
	resultObj.Set("delay", arena.NewNumberInt(delay))
	resultObj.Set("message", arena.NewString(fmt.Sprintf("Operation simulated with %d seconds delay", delay)))

	return resultObj, nil
}

// BatchDemo 批量操作演示
func (t *TestService) BatchDemo(arena *fastjson.Arena, params *fastjson.Value) (interface{}, error) {
	resultObj := arena.NewObject()
	resultObj.Set("message", arena.NewString("This is a batch operation demo"))
	resultObj.Set("timestamp", arena.NewString("2024-01-01T00:00:00Z"))

	statsArray := arena.NewArray()
	statsArray.SetArrayItem(0, arena.NewNumberInt(100))
	statsArray.SetArrayItem(1, arena.NewNumberInt(200))
	statsArray.SetArrayItem(2, arena.NewNumberInt(300))
	resultObj.Set("stats", statsArray)

	return resultObj, nil
}

// RegisterTestService 注册测试服务到 JSONRPC2 实例
func (j *JSONRPC2) RegisterTestService(prefix ...string) {
	testService := NewTestService()

	// 确定前缀
	servicePrefix := "test."
	if len(prefix) > 0 && prefix[0] != "" {
		servicePrefix = prefix[0]
	}

	// 注册测试服务
	err := j.RegisterObject(testService, servicePrefix)
	if err != nil {
		slog.Error(fmt.Sprintf("Warning: Failed to register test service: %v", err))
	}

	// 同时注册一些无前缀的常用方法
	j.RegisterMethodFunc("ping", func(params *fastjson.Value) (interface{}, error) {
		return testService.Ping(nil, params)
	})
	j.RegisterMethodFunc("echo", func(params *fastjson.Value) (interface{}, error) {
		return testService.Echo(nil, params)
	})

	slog.Info("Test service registered with prefix: ", servicePrefix)
}
