package bamgoo

import (
	"strconv"
	"sync"

	. "github.com/bamgoo/bamgoo/base"
)

const (
	START = "start"
	STOP  = "stop"
)

var (
	trigger = &triggerModule{
		triggers: make(map[string][]Trigger, 0),
		methods:  make(map[string][]string, 0),
	}
)

type (
	triggerModule struct {
		mutex    sync.Mutex
		triggers map[string][]Trigger
		methods  map[string][]string
		seq      uint64
	}
	Trigger struct {
		Name     string
		Desc     string
		Nullable bool
		Args     Vars
		Action   func(*Context)
	}
)

func (m *triggerModule) Register(name string, value Any) {
	if cfg, ok := value.(Trigger); ok {
		m.RegisterTrigger(name, cfg)
	}
}
func (m *triggerModule) RegisterTrigger(name string, cfg Trigger) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if name == "" {
		return
	}
	if _, ok := m.triggers[name]; !ok {
		m.triggers[name] = make([]Trigger, 0)
	}
	m.triggers[name] = append(m.triggers[name], cfg)
}

// Configure
func (m *triggerModule) Config(Map) {}

// Setup
func (m *triggerModule) Setup() {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	for name, triggers := range m.triggers {
		if _, ok := m.methods[name]; !ok {
			m.methods[name] = make([]string, 0)
		}
		for _, cfg := range triggers {
			methodName := m.nextMethodName(name)
			action := cfg.Action // capture for closure
			core.RegisterMethod(methodName, Method{
				Name: cfg.Name, Desc: cfg.Desc,
				Action: func(ctx *Context) (Map, Res) {
					action(ctx)
					return nil, nil
				},
			})
			m.methods[name] = append(m.methods[name], methodName)
		}
	}
}
func (m *triggerModule) Open() {}
func (m *triggerModule) Start() {
	m.Toggle(START)
}
func (m *triggerModule) Stop() {
	m.SyncToggle(STOP)
}
func (m *triggerModule) Close() {}

func (m *triggerModule) nextMethodName(name string) string {
	m.seq++
	return "_." + name + "." + strconv.FormatUint(m.seq, 10)
}

func (m *triggerModule) Toggle(name string, values ...Map) {
	value := Map{}
	if len(values) > 0 && values[0] != nil {
		value = values[0]
	}
	if ms, ok := m.methods[name]; ok {
		for _, methodName := range ms {
			go core.Invoke(nil, methodName, value)
		}
	}
}

func (m *triggerModule) SyncToggle(name string, values ...Map) {
	value := Map{}
	if len(values) > 0 && values[0] != nil {
		value = values[0]
	}
	if ms, ok := m.methods[name]; ok {
		for _, methodName := range ms {
			core.Invoke(nil, methodName, value)
		}
	}
}

func Toggle(name string, values ...Map) {
	trigger.Toggle(name, values...)
}

func SyncToggle(name string, values ...Map) {
	trigger.SyncToggle(name, values...)
}
