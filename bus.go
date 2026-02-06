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

		Request(subject string, data []byte, timeout time.Duration) ([]byte, error)
		Publish(subject string, data []byte) error
		Enqueue(subject string, data []byte) error

		Stats() []ServiceStats
	}

	// ServiceStats contains service statistics.
	ServiceStats struct {
		Name         string `json:"name"`
		Version      string `json:"version"`
		NumRequests  int    `json:"num_requests"`
		NumErrors    int    `json:"num_errors"`
		TotalLatency int64  `json:"total_latency_ms"`
		AvgLatency   int64  `json:"avg_latency_ms"`
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
	// busRequest combines metadata and payload for transmission.
	busRequest struct {
		Metadata
		Name    string `json:"name"`
		Payload Map    `json:"payload,omitempty"`
	}

	// busResponse contains result with full Res info.
	busResponse struct {
		Code  int    `json:"code"`
		State string `json:"state"`
		Desc  string `json:"desc,omitempty"`
		Time  int64  `json:"time"`
		Data  Map    `json:"data,omitempty"`
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

	data, err := encodeRequest(meta, name, value)
	if err != nil {
		return nil, errorResult(err)
	}

	base := m.subjectBase(prefix, name)
	subject := m.subject("", subjectCall, base)
	resBytes, err := conn.Request(subject, data, timeout)
	if err != nil {
		return nil, errorResult(err)
	}

	return decodeResponse(resBytes)
}

// Publish broadcasts an event to all subscribers.
func (m *busModule) Publish(meta *Meta, name string, value Map) error {
	conn, prefix := m.pick()

	if conn == nil {
		return errBusNotReady
	}

	data, err := encodeRequest(meta, name, value)
	if err != nil {
		return err
	}

	base := m.subjectBase(prefix, name)
	subject := m.subject("", subjectEvent, base)
	return conn.Publish(subject, data)
}

// Enqueue sends to a queue (one subscriber receives).
func (m *busModule) Enqueue(meta *Meta, name string, value Map) error {
	conn, prefix := m.pick()

	if conn == nil {
		return errBusNotReady
	}

	data, err := encodeRequest(meta, name, value)
	if err != nil {
		return err
	}

	base := m.subjectBase(prefix, name)
	subject := m.subject("", subjectQueue, base)
	return conn.Enqueue(subject, data)
}

func encodeRequest(meta *Meta, name string, payload Map) ([]byte, error) {
	req := busRequest{
		Name:    name,
		Payload: payload,
	}
	if meta != nil {
		req.Metadata = meta.Metadata()
	}
	return json.Marshal(req)
}

func decodeRequest(data []byte) (*Meta, string, Map, error) {
	var req busRequest
	if err := json.Unmarshal(data, &req); err != nil {
		return nil, "", nil, err
	}

	meta := NewMeta()
	meta.Metadata(req.Metadata)

	if req.Payload == nil {
		req.Payload = Map{}
	}
	return meta, req.Name, req.Payload, nil
}

func encodeResponse(data Map, res Res) ([]byte, error) {
	if res == nil {
		res = OK
	}
	resp := busResponse{
		Code:  res.Code(),
		State: res.State(),
		Desc:  res.Error(),
		Time:  time.Now().UnixMilli(),
		Data:  data,
	}
	return json.Marshal(resp)
}

func decodeResponse(data []byte) (Map, Res) {
	var resp busResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, errorResult(err)
	}

	res := Result(resp.Code, resp.State, resp.Desc)
	if resp.Data == nil {
		resp.Data = Map{}
	}
	return resp.Data, res
}

// HandleCall handles request/reply for a bus instance.
func (inst *BusInstance) HandleCall(data []byte) ([]byte, error) {
	meta, name, payload, err := decodeRequest(data)
	if err != nil {
		return nil, err
	}

	body, res, _ := core.invokeLocal(meta, name, payload)
	return encodeResponse(body, res)
}

// HandleAsync handles async execution (queue/event) for a bus instance.
func (inst *BusInstance) HandleAsync(data []byte) error {
	meta, name, payload, err := decodeRequest(data)
	if err != nil {
		return err
	}

	go core.invokeLocal(meta, name, payload)
	return nil
}

func Publish(name string, value Map) error {
	return bus.Publish(nil, name, value)
}

func Enqueue(name string, value Map) error {
	return bus.Enqueue(nil, name, value)
}

// Stats returns service statistics from all connections.
func Stats() []ServiceStats {
	return bus.Stats()
}

func (m *busModule) Stats() []ServiceStats {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	var all []ServiceStats
	for _, conn := range m.connections {
		stats := conn.Stats()
		if stats != nil {
			all = append(all, stats...)
		}
	}
	return all
}
