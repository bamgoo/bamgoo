package bamgoo

import (
	"fmt"
	"reflect"
	"sync"

	. "github.com/bamgoo/base"
)

var providers = &providerModule{
	providers: make(map[string]Provider),
}

// Provider defines a buildable provider template.
// UseProvider should return an instance safe for current invocation.
type Provider interface {
	UseProvider(setting Map) (Any, error)
}

type providerModule struct {
	mutex     sync.RWMutex
	providers map[string]Provider
}

func (m *providerModule) Register(name string, value Any) {
	if provider, ok := value.(Provider); ok {
		m.RegisterProvider(name, provider)
	}
}

func (m *providerModule) RegisterProvider(name string, provider Provider) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if name == "" {
		panic("provider name is required")
	}
	if provider == nil {
		panic("invalid provider: " + name)
	}

	if _, exists := m.providers[name]; exists && !Override() {
		panic("provider already registered: " + name)
	}

	m.providers[name] = provider
}

func (m *providerModule) Use(name string, setting Map) (Any, error) {
	m.mutex.RLock()
	provider, ok := m.providers[name]
	m.mutex.RUnlock()

	if !ok {
		return nil, fmt.Errorf("provider not registered: %s", name)
	}

	if setting == nil {
		setting = Map{}
	}

	impl, err := provider.UseProvider(setting)
	if err != nil {
		return nil, fmt.Errorf("build provider failed: %s: %w", name, err)
	}
	if impl == nil {
		return nil, fmt.Errorf("provider build returned nil: %s", name)
	}

	return impl, nil
}

func (m *providerModule) Config(Map) {}
func (m *providerModule) Setup()     {}
func (m *providerModule) Open()      {}
func (m *providerModule) Start()     {}
func (m *providerModule) Stop()      {}
func (m *providerModule) Close()     {}

// Use resolves a provider by name and builds a typed instance with setting.
func Use[T any](name string, setting Map) (T, error) {
	var zero T

	impl, err := providers.Use(name, setting)
	if err != nil {
		return zero, err
	}

	typed, ok := impl.(T)
	if !ok {
		want := reflect.TypeOf((*T)(nil)).Elem()
		return zero, fmt.Errorf("provider type mismatch: %s want=%v got=%T", name, want, impl)
	}

	return typed, nil
}
