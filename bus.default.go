package bamgoo

import (
	"encoding/json"
	"errors"
	"sync"
	"time"

	. "github.com/bamgoo/bamgoo/base"
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
	}
)

// Connect establishes an in-memory bus.
func (driver *defaultBusDriver) Connect(_ *BusInstance) (Connection, error) {
	return &defaultBusConnection{
		services: make(map[string]struct{}, 0),
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
func (c *defaultBusConnection) Request(meta *Meta, name string, data []byte, _ time.Duration) ([]byte, error) {
	name, payload, err := c.decodeRequest(data)
	if err != nil {
		return nil, err
	}
	body, res, _ := core.invokeLocal(meta, name, payload)
	return c.encodeResponse(body, res)
}

// Publish broadcasts event to all local handlers - for in-memory, invoke local.
func (c *defaultBusConnection) Publish(meta *Meta, name string, data []byte) error {
	name, payload, err := c.decodeRequest(data)
	if err != nil {
		return err
	}
	go core.invokeLocal(meta, name, payload)
	return nil
}

// Queue handles queued call - for in-memory bus, directly invoke local.
func (c *defaultBusConnection) Enqueue(meta *Meta, name string, data []byte) error {
	name, payload, err := c.decodeRequest(data)
	if err != nil {
		return err
	}
	go core.invokeLocal(meta, name, payload)
	return nil
}

func (c *defaultBusConnection) decodeRequest(data []byte) (string, Map, error) {
	var env struct {
		Name    string `json:"name"`
		Payload Map    `json:"payload"`
	}
	if err := json.Unmarshal(data, &env); err != nil {
		return "", nil, err
	}
	if env.Payload == nil {
		env.Payload = Map{}
	}
	return env.Name, env.Payload, nil
}

func (c *defaultBusConnection) encodeResponse(data Map, res Res) ([]byte, error) {
	if res == nil {
		res = OK
	}
	env := struct {
		Code    int    `json:"code"`
		State   string `json:"state"`
		Message string `json:"message"`
		Data    Map    `json:"data"`
	}{
		Code:    res.Code(),
		State:   res.State(),
		Message: res.Error(),
		Data:    data,
	}
	return json.Marshal(env)
}
