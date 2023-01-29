package fastjsonrpc

import (
	"errors"
	"go/token"
	"reflect"
	"strings"
	"sync"
)

var typeOfContext = reflect.TypeOf(&Context{})

type service struct {
	name   string             // name of service
	rcvr   reflect.Value      // receiver of methods for the service
	typ    reflect.Type       // type of the receiver
	method map[string]Handler // registered methods
}

type ServerMap struct {
	serviceMap sync.Map // map[string]*service
}

func (p *ServerMap) Register(rcvr any) error {
	return p.register(rcvr, "", false)
}

func (p *ServerMap) RegisterName(name string, rcvr any) error {
	return p.register(rcvr, name, true)
}
func (p *ServerMap) RegisterHandler(method string, handler Handler) {
	var s *service
	t, ok := p.serviceMap.Load("~")
	if !ok {
		s = new(service)
		s.method = make(map[string]Handler)
		p.serviceMap.Store("~", s)
	} else {
		s = t.(*service)
	}
	s.method[method] = handler
}
func (p *ServerMap) register(rcvr any, name string, useName bool) error {
	s := new(service)
	s.typ = reflect.TypeOf(rcvr)
	s.rcvr = reflect.ValueOf(rcvr)
	sname := name
	if !useName {
		sname = reflect.Indirect(s.rcvr).Type().Name()
	}
	if sname == "" {
		return errors.New("rpc.Register: no service name for type " + s.typ.String())
	}
	if !useName && !token.IsExported(sname) {
		return errors.New("rpc.Register: type " + sname + " is not exported")
	}
	s.name = sname

	s.method = suitableMethods(s)

	if _, dup := p.serviceMap.LoadOrStore(sname, s); dup {
		return errors.New("rpc: service already defined: " + sname)
	}
	return nil
}

func (p *ServerMap) getFun(m string) (h Handler) {
	var serviceName, methodName string
	dot := strings.LastIndex(m, ".")
	if dot < 0 {
		serviceName = "~"
		methodName = m
	} else {
		serviceName = m[:dot]
		methodName = m[dot+1:]
	}

	s, ok := p.serviceMap.Load(serviceName)
	if !ok {
		return
	}
	svc := s.(*service)
	h = svc.method[methodName]
	return
}

func suitableMethods(s *service) map[string]Handler {
	methods := make(map[string]Handler)
	for m := 0; m < s.typ.NumMethod(); m++ {
		method := s.typ.Method(m)

		if method.Type.NumIn() != 2 {
			continue
		}

		argType := method.Type.In(1)
		if argType != typeOfContext {
			continue
		}

		methods[method.Name] = func(c *Context) { method.Func.Call([]reflect.Value{s.rcvr, reflect.ValueOf(c)}) }
	}
	return methods
}
