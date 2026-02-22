package bamgoo

import (
	"strings"
	"sync"

	. "github.com/bamgoo/base"
)

var library = &libraryModule{
	libraries: make(map[string]Library),
}

type (
	// Library defines a method group with defaults.
	// Example:
	// Register("mail.sendcloud", Library{
	//   Name: "SendCloud",
	//   Methods: Methods{"send": Method{...}},
	// })
	Library struct {
		Name    string
		Desc    string
		Setting Map
		Methods Methods
	}
)

type libraryModule struct {
	mutex     sync.RWMutex
	libraries map[string]Library
}

type libraryInvoker struct {
	meta    *Meta
	name    string
	setting Map
	result  Res
}

func (m *libraryModule) Register(name string, value Any) {
	switch v := value.(type) {
	case Library:
		m.RegisterLibrary(name, v)
	}
}

func (m *libraryModule) RegisterLibrary(prefix string, def Library) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	prefix = normalizeLibraryName(prefix)
	if prefix == "" || len(def.Methods) == 0 {
		return
	}

	if def.Setting == nil {
		def.Setting = Map{}
	}

	if def.Name == "" {
		def.Name = prefix
	}
	m.libraries[prefix] = def

	for key, method := range def.Methods {
		key = normalizeLibraryName(key)
		if key == "" {
			continue
		}

		full := joinLibraryName(prefix, key)
		core.RegisterMethod(full, method)
	}
}

func (m *libraryModule) Config(Map) {}
func (m *libraryModule) Setup()     {}
func (m *libraryModule) Open() {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if m.libraries == nil {
		m.libraries = make(map[string]Library)
	}
}
func (m *libraryModule) Start() {}
func (m *libraryModule) Stop()  {}
func (m *libraryModule) Close() {}

func (m *Meta) Library(name string, settings ...Map) *libraryInvoker {
	name = normalizeLibraryName(name)

	setting := Map{}
	if libDef, ok := library.Load(name); ok {
		for k, v := range libDef.Setting {
			setting[k] = v
		}
	}
	for _, item := range settings {
		for k, v := range item {
			setting[k] = v
		}
	}
	return &libraryInvoker{
		meta:    m,
		name:    name,
		setting: setting,
	}
}

func (l *libraryInvoker) Invoke(name string, values ...Map) Map {
	if l == nil {
		return nil
	}
	value := Map{}
	if len(values) > 0 && values[0] != nil {
		value = values[0]
	}

	fullName := joinLibraryName(l.name, normalizeLibraryName(name))
	data, res, ok := core.invokeLocal(l.meta, fullName, value, l.setting)
	if !ok {
		res = textResult("library method not found: " + fullName)
	}
	l.result = res
	l.meta.Result(res)
	return data
}

func (l *libraryInvoker) Result() Res {
	if l == nil {
		return OK
	}
	if l.result == nil {
		return OK
	}
	res := l.result
	l.result = nil
	return res
}

func (m *libraryModule) Load(name string) (Library, bool) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	if m.libraries == nil {
		return Library{}, false
	}
	lib, ok := m.libraries[name]
	return lib, ok
}

func normalizeLibraryName(name string) string {
	name = strings.TrimSpace(strings.ToLower(name))
	name = strings.TrimPrefix(name, ".")
	name = strings.TrimSuffix(name, ".")
	return name
}

func joinLibraryName(prefix, key string) string {
	if prefix == "" {
		return key
	}
	if key == "" {
		return prefix
	}
	return prefix + "." + key
}
