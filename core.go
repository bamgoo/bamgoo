package bamgoo

import (
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	. "github.com/bamgoo/bamgoo/base"
)

var core = &coreModule{
	entries: make(map[string]coreEntry, 0),
}

const defaultCallTimeout = 5 * time.Second

type (
	coreModule struct {
		mutex   sync.RWMutex
		entries map[string]coreEntry
	}
	coreEntry struct {
		remote bool

		Name     string
		Desc     string
		Nullable bool
		Args     Vars
		Action   func(*Context) (Map, Res)
		Setting  Map
	}
)
type (
	Method struct {
		Name     string
		Desc     string
		Nullable bool
		Args     Vars
		Action   func(*Context) (Map, Res)
		Setting  Map
	}
	Service struct {
		Name     string
		Desc     string
		Nullable bool
		Args     Vars
		Action   func(*Context) (Map, Res)
		Setting  Map
	}
)

func (e *coreModule) Register(name string, value Any) {
	switch v := value.(type) {
	case Method:
		e.RegisterMethod(name, v)
	case Service:
		e.RegisterService(name, v)
	}
}

func (e *coreModule) RegisterMethod(name string, method Method) {
	e.mutex.Lock()
	defer e.mutex.Unlock()

	if name == "" {
		return
	}

	if _, ok := e.entries[name]; ok {
		panic("method already registered: " + name)
	}

	e.entries[name] = coreEntry{
		remote:   false,
		Name:     name,
		Desc:     method.Desc,
		Nullable: method.Nullable,
		Args:     method.Args,
		Action:   method.Action,
		Setting:  method.Setting,
	}
}

func (e *coreModule) RegisterService(name string, service Service) {
	e.mutex.Lock()
	defer e.mutex.Unlock()

	if name == "" {
		return
	}
	if _, ok := e.entries[name]; ok {
		panic("service already registered: " + name)
	}
	e.entries[name] = coreEntry{
		remote:   true,
		Name:     name,
		Desc:     service.Desc,
		Nullable: service.Nullable,
		Args:     service.Args,
		Action:   service.Action,
	}
}

func (e *coreModule) Config(Map) {}
func (e *coreModule) Setup()     {}
func (e *coreModule) Open()      {}
func (e *coreModule) Start() {
	fmt.Println("core module is running.")
}
func (e *coreModule) Stop()  {}
func (e *coreModule) Close() {}

func (e *coreModule) Wait() {
	// 待处理，加入自己的退出信号
	// 并开放 infra.Stop() 给外部调用
	waiter := make(chan os.Signal, 1)
	signal.Notify(waiter, os.Kill, os.Interrupt, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	<-waiter
}

// Invoke calls a method/service (local first, then remote via bus).
func (e *coreModule) Invoke(meta *Meta, name string, value Map) (Map, Res) {
	if data, res, ok := e.invokeLocal(meta, name, value); ok {
		return data, res
	}
	return e.invokeRemote(meta, name, value)
}

// localInvoke only calls local method/service, does not go through bus.
// Returns (data, res, found) where found indicates if local entry exists.
func (e *coreModule) invokeLocal(meta *Meta, name string, value Map) (Map, Res, bool) {
	e.mutex.RLock()
	entry, ok := e.entries[name]
	e.mutex.RUnlock()

	if !ok || entry.Action == nil {
		return nil, nil, false
	}

	if meta == nil {
		meta = NewMeta()
	}
	ctx := &Context{
		Meta:    meta,
		Name:    name,
		Config:  &entry,
		Setting: Map{},
		Value:   value,
	}
	for k, v := range entry.Setting {
		ctx.Setting[k] = v
	}
	data, res := entry.Action(ctx)
	return data, res, true
}

// remoteInvoke calls remote service via bus.
func (e *coreModule) invokeRemote(meta *Meta, name string, value Map) (Map, Res) {
	if meta == nil {
		meta = NewMeta()
	}
	return bus.Request(meta, name, value, defaultCallTimeout)
}
