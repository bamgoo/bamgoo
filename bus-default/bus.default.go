package bus_default

import (
	"errors"
	"sync"
	"time"

	"github.com/bamgoo/bamgoo"
	"github.com/bamgoo/bamgoo/bus"
)

var (
	errBusRunning       = errors.New("bus is running")
	errBusNotRunning    = errors.New("bus is not running")
	errBusInvalidTarget = errors.New("invalid bus target")
)

type (
	defaultBusDriver struct{}

	defaultBusConnection struct {
		mutex    sync.RWMutex
		running  bool
		services map[string]struct{}
		instance *bus.BusInstance
	}
)

func init() {
	bamgoo.Register(bamgoo.DEFAULT, &defaultBusDriver{})
}

// Connect establishes an in-memory bus.
func (driver *defaultBusDriver) Connect(inst *bus.BusInstance) (bus.Connection, error) {
	return &defaultBusConnection{
		services: make(map[string]struct{}, 0),
		instance: inst,
	}, nil
}

func (c *defaultBusConnection) Open() error  { return nil }
func (c *defaultBusConnection) Close() error { return nil }

func (c *defaultBusConnection) Start() error {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	if c.running {
		return errBusRunning
	}
	c.running = true
	return nil
}

func (c *defaultBusConnection) Stop() error {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	if !c.running {
		return errBusNotRunning
	}
	c.running = false
	return nil
}

// Register registers a service subject for local handling.
func (c *defaultBusConnection) Register(subject string) error {
	if subject == "" {
		return errBusInvalidTarget
	}

	c.mutex.Lock()
	c.services[subject] = struct{}{}
	c.mutex.Unlock()
	return nil
}

// Request handles synchronous call - for in-memory bus, directly invoke local.
func (c *defaultBusConnection) Request(_ string, data []byte, _ time.Duration) ([]byte, error) {
	if c.instance == nil {
		c.instance = &bus.BusInstance{}
	}
	return c.instance.HandleCall(data)
}

// Publish broadcasts event to all local handlers - for in-memory, invoke local.
func (c *defaultBusConnection) Publish(_ string, data []byte) error {
	if c.instance == nil {
		c.instance = &bus.BusInstance{}
	}
	return c.instance.HandleAsync(data)
}

// Enqueue handles queued call - for in-memory bus, directly invoke local.
func (c *defaultBusConnection) Enqueue(_ string, data []byte) error {
	if c.instance == nil {
		c.instance = &bus.BusInstance{}
	}
	return c.instance.HandleAsync(data)
}

// Stats returns empty stats for in-memory bus.
func (c *defaultBusConnection) Stats() []bamgoo.ServiceStats {
	return nil
}
