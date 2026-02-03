package bamgoo

import (
	"encoding/json"
	"errors"
	"sync"
	"time"

	. "github.com/bamgoo/bamgoo/base"
	"github.com/bamgoo/bamgoo/util"
)

var (
	errBusNotReady = errors.New("bus is not ready")
)

var (
	bus = &busModule{
		drivers:     make(map[string]Driver, 0),
		configs:     make(map[string]BusConfig, 0),
		connections: make(map[string]Connection, 0),
		weights:     make(map[string]int, 0),
		services:    make(map[string]struct{}, 0),
	}
)

type (
	// Handler processes incoming payload and returns reply bytes for call.
	Handler func([]byte) ([]byte, error)

	// Driver connections a bus transport.
	Driver interface {
		Connect(*BusInstance) (Connection, error)
	}

	// Connect defines a bus transport connection.
	Connection interface {
		Open() error
		Close() error
		Start() error
		Stop() error

		Register(subject string) error

		Request(*Meta, string, []byte, time.Duration) ([]byte, error)
		Publish(*Meta, string, []byte) error
		Enqueue(*Meta, string, []byte) error
	}

	busModule struct {
		mutex sync.RWMutex

		drivers     map[string]Driver
		configs     map[string]BusConfig
		connections map[string]Connection
		weights     map[string]int
		wrr         *util.WRR
		services    map[string]struct{}

		opened  bool
		started bool
	}

	BusInstance struct {
		Name   string
		Config BusConfig
	}

	BusConfig struct {
		Driver  string
		Weight  int
		Prefix  string
		Setting Map
	}

	Configs map[string]BusConfig
)

const (
	subjectCall  = "call"
	subjectQueue = "queue"
	subjectEvent = "event"
)

type (
	requestEnvelope struct {
		Name    string `json:"name"`
		Payload Map    `json:"payload"`
	}
	responseEnvelope struct {
		Code    int    `json:"code"`
		State   string `json:"state"`
		Message string `json:"message"`
		Data    Map    `json:"data"`
	}
)

// Register dispatches registrations.
func (m *busModule) Register(name string, value Any) {
	switch v := value.(type) {
	case Driver:
		m.RegisterDriver(name, v)
	case BusConfig:
		m.RegisterConfig(name, v)
	case Configs:
		m.RegisterConfigs(v)
	case Service:
		m.RegisterService(name)
	}
}

// RegisterDriver registers a bus driver.
func (m *busModule) RegisterDriver(name string, driver Driver) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if name == "" {
		name = DEFAULT
	}
	if driver == nil {
		panic("Invalid bus driver: " + name)
	}
	if _, ok := m.drivers[name]; ok {
		panic("Bus driver already registered: " + name)
	}
	m.drivers[name] = driver
}

// RegisterConfig registers a named bus config.
// If name is empty, it uses DEFAULT.
func (m *busModule) RegisterConfig(name string, cfg BusConfig) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if m.opened || m.started {
		return
	}

	if name == "" {
		name = DEFAULT
	}
	if _, ok := m.configs[name]; ok {
		panic("Bus config already registered: " + name)
	}
	m.configs[name] = cfg
}

// RegisterConfigs registers multiple named bus configs.
func (m *busModule) RegisterConfigs(configs Configs) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if m.opened || m.started {
		return
	}

	for name, cfg := range configs {
		if name == "" {
			name = DEFAULT
		}
		if _, ok := m.configs[name]; ok {
			panic("Bus config already registered: " + name)
		}
		m.configs[name] = cfg
	}
}

// RegisterService binds service name into bus subjects.
func (m *busModule) RegisterService(name string) {
	if name == "" {
		return
	}
	m.mutex.Lock()
	m.services[name] = struct{}{}
	m.mutex.Unlock()
}

func (m *busModule) Config(_ Map) {}

// Setup initializes defaults.
func (m *busModule) Setup() {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if m.opened || m.started {
		return
	}

	if len(m.configs) == 0 {
		m.configs[DEFAULT] = BusConfig{Driver: DEFAULT, Weight: 1}
	}

	// normalize configs
	for name, cfg := range m.configs {
		if name == "" {
			name = DEFAULT
		}
		if cfg.Driver == "" {
			cfg.Driver = DEFAULT
		}
		if cfg.Weight == 0 {
			cfg.Weight = 1
		}
		m.configs[name] = cfg
	}
}

// Open connections bus and registers services.
func (m *busModule) Open() {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if m.opened {
		return
	}

	if len(m.configs) == 0 {
		panic("Missing bus config")
	}

	for name, cfg := range m.configs {
		driver, ok := m.drivers[cfg.Driver]
		if !ok || driver == nil {
			panic("Missing bus driver: " + cfg.Driver)
		}

		if cfg.Weight == 0 {
			cfg.Weight = 1
		}

		inst := &BusInstance{Name: name, Config: cfg}
		conn, err := driver.Connect(inst)
		if err != nil {
			panic("Failed to connect to bus: " + err.Error())
		}
		if err := conn.Open(); err != nil {
			panic("Failed to open bus: " + err.Error())
		}

		for svc := range m.services {
			base := m.subjectBase(cfg.Prefix, svc)
			if err := conn.Register(base); err != nil {
				panic("Failed to register bus: " + err.Error())
			}
		}

		m.connections[name] = conn
		m.weights[name] = cfg.Weight
	}

	m.wrr = util.NewWRR(m.weights)
	m.opened = true
}

// Start launches bus subscriptions.
func (m *busModule) Start() {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if m.started {
		return
	}

	if len(m.connections) == 0 {
		panic("Bus not opened")
	}

	for _, conn := range m.connections {
		_ = conn.Start()
	}
	m.started = true
}

// Stop terminates bus subscriptions.
func (m *busModule) Stop() {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if !m.started {
		return
	}

	for _, conn := range m.connections {
		_ = conn.Stop()
	}

	m.started = false
}

// Close closes bus connections.
func (m *busModule) Close() {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if !m.opened {
		return
	}

	for _, conn := range m.connections {
		conn.Close()
	}

	m.connections = make(map[string]Connection, 0)
	m.weights = make(map[string]int, 0)
	m.wrr = nil
	m.opened = false
}

func (m *busModule) subject(prefix, kind, name string) string {
	if prefix == "" {
		return kind + "." + name
	}
	return prefix + kind + "." + name
}

func (m *busModule) subjectBase(prefix, name string) string {
	if prefix == "" {
		return name
	}
	return prefix + name
}

func (m *busModule) pick() (Connection, string) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if m.wrr == nil {
		return nil, ""
	}
	name := m.wrr.Next()
	if name == "" {
		return nil, ""
	}
	conn := m.connections[name]
	cfg := m.configs[name]
	return conn, cfg.Prefix
}

// Request sends a request and waits for reply.
func (m *busModule) Request(meta *Meta, name string, value Map, timeout time.Duration) (Map, Res) {
	conn, prefix := m.pick()

	if conn == nil {
		return nil, errorResult(errBusNotReady)
	}

	payload, err := encodeRequest(name, value)
	if err != nil {
		return nil, errorResult(err)
	}

	base := m.subjectBase(prefix, name)
	subject := m.subject("", subjectCall, base)
	resBytes, err := conn.Request(meta, subject, payload, timeout)
	if err != nil {
		return nil, errorResult(err)
	}

	data, res, err := decodeResponse(resBytes)
	if err != nil {
		return nil, errorResult(err)
	}

	return data, res
}

// Cast publishes a queue call.
func (m *busModule) Publish(meta *Meta, name string, value Map) error {
	conn, prefix := m.pick()

	if conn == nil {
		return errBusNotReady
	}

	payload, err := encodeRequest(name, value)
	if err != nil {
		return err
	}

	base := m.subjectBase(prefix, name)
	subject := m.subject("", subjectQueue, base)
	return conn.Enqueue(meta, subject, payload)
}

// Emit publishes an event call.
func (m *busModule) Enqueue(meta *Meta, name string, value Map) error {
	conn, prefix := m.pick()

	if conn == nil {
		return errBusNotReady
	}

	payload, err := encodeRequest(name, value)
	if err != nil {
		return err
	}

	base := m.subjectBase(prefix, name)
	subject := m.subject("", subjectEvent, base)
	return conn.Enqueue(meta, subject, payload)
}

func encodeRequest(name string, payload Map) ([]byte, error) {
	return json.Marshal(requestEnvelope{Name: name, Payload: payload})
}

func decodeRequest(data []byte) (string, Map, error) {
	var env requestEnvelope
	if err := json.Unmarshal(data, &env); err != nil {
		return "", nil, err
	}
	if env.Payload == nil {
		env.Payload = Map{}
	}
	return env.Name, env.Payload, nil
}

func encodeResponse(data Map, res Res) ([]byte, error) {
	if res == nil {
		res = OK
	}
	env := responseEnvelope{
		Code:    res.Code(),
		State:   res.State(),
		Message: res.Error(),
		Data:    data,
	}
	return json.Marshal(env)
}

func decodeResponse(data []byte) (Map, Res, error) {
	var env responseEnvelope
	if err := json.Unmarshal(data, &env); err != nil {
		return nil, nil, err
	}
	state := env.State
	if state == "" {
		state = env.Message
	}
	res := Result(env.Code, state, env.Message)
	if env.Data == nil {
		env.Data = Map{}
	}
	return env.Data, res, nil
}
