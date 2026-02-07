package bamgoo

import (
	"errors"
	"sync"
	"time"

	base "github.com/bamgoo/bamgoo/base"
)

var (
	errBusBridgeMissing    = errors.New("bus bridge not registered")
	errConfigBridgeMissing = errors.New("config bridge not registered")
)

// Bridge exposes bridge registrations and access.
var bridge = &bridgeModule{}

type (
	bridgeModule struct {
		mutex sync.RWMutex

		config ConfigBridge
		bus    BusBridge
	}

	ConfigBridge interface {
		LoadConfig() (base.Map, error)
	}

	BusBridge interface {
		Request(meta *Meta, name string, value base.Map, timeout time.Duration) (base.Map, base.Res)
		Publish(meta *Meta, name string, value base.Map) error
		Enqueue(meta *Meta, name string, value base.Map) error
		Stats() []ServiceStats
	}
)

// Register dispatches Module.Register based on type.
func (b *bridgeModule) Register(name string, value base.Any) {
	switch v := value.(type) {
	case BusBridge:
		b.RegisterBus(v)
	case ConfigBridge:
		b.RegisterConfig(v)
	}
}

func (b *bridgeModule) RegisterBus(bridge BusBridge) {
	b.mutex.Lock()
	defer b.mutex.Unlock()

	if bridge == nil {
		panic("Invalid bus bridge")
	}
	if b.bus != nil {
		panic("Bus bridge already registered")
	}

	b.bus = bridge
}

func (b *bridgeModule) RegisterConfig(bridge ConfigBridge) {
	b.mutex.Lock()
	defer b.mutex.Unlock()

	if bridge == nil {
		panic("Invalid config bridge")
	}
	if b.config != nil {
		panic("Config bridge already registered")
	}

	b.config = bridge
}

// Module methods
func (b *bridgeModule) Config(base.Map) {}
func (b *bridgeModule) Setup()          {}
func (b *bridgeModule) Open()           {}
func (b *bridgeModule) Start()          {}
func (b *bridgeModule) Stop()           {}
func (b *bridgeModule) Close()          {}

func (b *bridgeModule) LoadConfig() (base.Map, error) {
	b.mutex.RLock()
	defer b.mutex.RUnlock()

	if b.config == nil {
		return nil, errConfigBridgeMissing
	}
	return b.config.LoadConfig()
}

func (b *bridgeModule) Publish(name string, value base.Map) error {
	b.mutex.RLock()
	defer b.mutex.RUnlock()
	if b.bus == nil {
		return errBusBridgeMissing
	}
	return b.bus.Publish(nil, name, value)
}

func (b *bridgeModule) Enqueue(name string, value base.Map) error {
	b.mutex.RLock()
	defer b.mutex.RUnlock()
	if b.bus == nil {
		return errBusBridgeMissing
	}
	return b.bus.Enqueue(nil, name, value)
}

func (b *bridgeModule) Stats() []ServiceStats {
	b.mutex.RLock()
	defer b.mutex.RUnlock()
	if b.bus == nil {
		return nil
	}
	return b.bus.Stats()
}

// Request performs a bus request.
func (b *bridgeModule) Request(meta *Meta, name string, value base.Map, timeout time.Duration) (base.Map, base.Res) {
	b.mutex.RLock()
	defer b.mutex.RUnlock()
	if b.bus == nil {
		return nil, ErrorResult(errBusBridgeMissing)
	}
	return b.bus.Request(meta, name, value, timeout)
}
