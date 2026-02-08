package bamgoo

import (
	"errors"
	"sync"
	"time"

	base "github.com/bamgoo/base"
)

var (
	errBusHookMissing    = errors.New("bus hook not registered")
	errConfigHookMissing = errors.New("config hook not registered")
)

// Hook exposes hook registrations and access (main -> sub).
var hook = &bamgooHook{}

type (
	bamgooHook struct {
		mutex sync.RWMutex

		bus    BusHook
		config ConfigHook
	}

	BusHook interface {
		Request(meta *Meta, name string, value base.Map, timeout time.Duration) (base.Map, base.Res)
		Publish(meta *Meta, name string, value base.Map) error
		Enqueue(meta *Meta, name string, value base.Map) error
		Stats() []ServiceStats
	}

	ConfigHook interface {
		LoadConfig() (base.Map, error)
	}
)

// Attach dispatches Module.Attach based on type.
func (h *bamgooHook) Attach(value base.Any) {
	switch v := value.(type) {
	case BusHook:
		h.AttachBus(v)
	case ConfigHook:
		h.AttachConfig(v)
	}
}

func (h *bamgooHook) AttachBus(hook BusHook) {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	if hook == nil {
		panic("Invalid bus hook")
	}

	h.bus = hook
}

func (h *bamgooHook) AttachConfig(hook ConfigHook) {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	if hook == nil {
		panic("Invalid config hook")
	}

	h.config = hook
}

func (h *bamgooHook) LoadConfig() (base.Map, error) {
	h.mutex.RLock()
	defer h.mutex.RUnlock()

	if h.config == nil {
		return nil, errConfigHookMissing
	}
	return h.config.LoadConfig()
}

// Request sends a bus request (main -> sub).
func (h *bamgooHook) Request(meta *Meta, name string, value base.Map, timeout time.Duration) (base.Map, base.Res) {
	h.mutex.RLock()
	defer h.mutex.RUnlock()
	if h.bus == nil {
		return nil, ErrorResult(errBusHookMissing)
	}
	return h.bus.Request(meta, name, value, timeout)
}

func (h *bamgooHook) Publish(name string, value base.Map) error {
	h.mutex.RLock()
	defer h.mutex.RUnlock()
	if h.bus == nil {
		return errBusHookMissing
	}
	return h.bus.Publish(nil, name, value)
}

func (h *bamgooHook) Enqueue(name string, value base.Map) error {
	h.mutex.RLock()
	defer h.mutex.RUnlock()
	if h.bus == nil {
		return errBusHookMissing
	}
	return h.bus.Enqueue(nil, name, value)
}

func (h *bamgooHook) Stats() []ServiceStats {
	h.mutex.RLock()
	defer h.mutex.RUnlock()
	if h.bus == nil {
		return nil
	}
	return h.bus.Stats()
}
