package http

import (
	"sync"
	"time"

	"github.com/bamgoo/bamgoo"
	. "github.com/bamgoo/base"
)

func init() {
	bamgoo.Mount(module)
}

var module = &Module{
	config: Config{
		Driver: DEFAULT, Charset: UTF8, Port: 8080,
	},
	cross: Cross{
		Allow: true,
	},
	drivers:  make(map[string]Driver, 0),
	routers:  make(map[string]Router, 0),
	filters:  make(map[string]Filter, 0),
	handlers: make(map[string]Handler, 0),
}

type (
	Module struct {
		mutex    sync.Mutex
		instance *Instance

		connected, initialized, launched bool

		config  Config
		cross   Cross
		drivers map[string]Driver

		routers  map[string]Router
		filters  map[string]Filter
		handlers map[string]Handler

		routerInfos map[string]Info

		serveFilters    []ctxFunc
		requestFilters  []ctxFunc
		executeFilters  []ctxFunc
		responseFilters []ctxFunc

		foundHandlers  []ctxFunc
		errorHandlers  []ctxFunc
		failedHandlers []ctxFunc
		deniedHandlers []ctxFunc
	}

	Config struct {
		Driver string
		Port   int
		Host   string

		CertFile string
		KeyFile  string

		Charset string

		Cookie   string
		Token    bool
		Expire   time.Duration
		Crypto   bool
		MaxAge   time.Duration
		HttpOnly bool

		Upload   string
		Static   string
		Defaults []string

		Setting Map
	}

	Cross struct {
		Allow   bool
		Method  string
		Methods []string
		Origin  string
		Origins []string
		Header  string
		Headers []string
	}

	Instance struct {
		connect Connect
		Config  Config
		Setting Map
	}
)

// Register dispatches registrations.
func (m *Module) Register(name string, value Any) {
	switch v := value.(type) {
	case Driver:
		m.RegisterDriver(name, v)
	case Config:
		m.RegisterConfig(v)
	case Router:
		m.RegisterRouter(name, v)
	case Filter:
		m.RegisterFilter(name, v)
	case Handler:
		m.RegisterHandler(name, v)
	}
}

// RegisterDriver registers an HTTP driver.
func (m *Module) RegisterDriver(name string, driver Driver) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if driver == nil {
		panic("Invalid http driver: " + name)
	}
	if name == "" {
		name = DEFAULT
	}

	if bamgoo.Override() {
		m.drivers[name] = driver
	} else {
		if _, ok := m.drivers[name]; !ok {
			m.drivers[name] = driver
		}
	}
}

// RegisterConfig registers HTTP config.
func (m *Module) RegisterConfig(config Config) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if m.initialized {
		return
	}
	m.config = config
}

func (m *Module) Config(Map) {}

func (m *Module) Setup() {
	if m.initialized {
		return
	}

	// Apply defaults
	if m.config.Port <= 0 || m.config.Port > 65535 {
		m.config.Port = 0
	}
	if m.config.Charset == "" {
		m.config.Charset = UTF8
	}
	if m.config.Defaults == nil || len(m.config.Defaults) == 0 {
		m.config.Defaults = []string{"index.html", "default.html"}
	}
	if m.config.Expire == 0 {
		m.config.Expire = time.Hour * 24 * 30
	}
	if m.config.MaxAge == 0 {
		m.config.MaxAge = time.Hour * 24 * 30
	}

	// Initialize router infos
	m.routerInfos = make(map[string]Info, 0)
	for key, router := range m.routers {
		for i, uri := range router.Uris {
			infoKey := key
			if i > 0 {
				infoKey = key + "." + string(rune('0'+i))
			}
			m.routerInfos[infoKey] = Info{
				Method: router.Method,
				Uri:    uri,
				Router: key,
				Args:   router.Args,
			}
		}
	}

	// Initialize filters
	m.serveFilters = make([]ctxFunc, 0)
	m.requestFilters = make([]ctxFunc, 0)
	m.executeFilters = make([]ctxFunc, 0)
	m.responseFilters = make([]ctxFunc, 0)

	for _, filter := range m.filters {
		if filter.Serve != nil {
			m.serveFilters = append(m.serveFilters, filter.Serve)
		}
		if filter.Request != nil {
			m.requestFilters = append(m.requestFilters, filter.Request)
		}
		if filter.Execute != nil {
			m.executeFilters = append(m.executeFilters, filter.Execute)
		}
		if filter.Response != nil {
			m.responseFilters = append(m.responseFilters, filter.Response)
		}
	}

	// Initialize handlers
	m.foundHandlers = make([]ctxFunc, 0)
	m.errorHandlers = make([]ctxFunc, 0)
	m.failedHandlers = make([]ctxFunc, 0)
	m.deniedHandlers = make([]ctxFunc, 0)

	for _, handler := range m.handlers {
		if handler.Found != nil {
			m.foundHandlers = append(m.foundHandlers, handler.Found)
		}
		if handler.Error != nil {
			m.errorHandlers = append(m.errorHandlers, handler.Error)
		}
		if handler.Failed != nil {
			m.failedHandlers = append(m.failedHandlers, handler.Failed)
		}
		if handler.Denied != nil {
			m.deniedHandlers = append(m.deniedHandlers, handler.Denied)
		}
	}

	m.initialized = true
}

func (m *Module) Open() {
	if m.connected {
		return
	}

	driver, ok := m.drivers[m.config.Driver]
	if !ok {
		panic("Invalid http driver: " + m.config.Driver)
	}

	inst := &Instance{
		Config:  m.config,
		Setting: m.config.Setting,
	}

	connect, err := driver.Connect(inst)
	if err != nil {
		panic("Failed to connect http: " + err.Error())
	}

	if err := connect.Open(); err != nil {
		panic("Failed to open http: " + err.Error())
	}

	// Register routes
	for name, info := range m.routerInfos {
		if err := connect.Register(name, info); err != nil {
			panic("Failed to register http route: " + err.Error())
		}
	}

	inst.connect = connect
	m.instance = inst
	m.connected = true
}

func (m *Module) Start() {
	if m.launched {
		return
	}

	if m.config.Port > 0 && m.config.Port < 65535 {
		var err error
		if m.config.CertFile != "" && m.config.KeyFile != "" {
			err = m.instance.connect.StartTLS(m.config.CertFile, m.config.KeyFile)
		} else {
			err = m.instance.connect.Start()
		}
		if err != nil {
			panic("Failed to start http: " + err.Error())
		}
	}

	m.launched = true
}

func (m *Module) Stop() {
	if !m.launched {
		return
	}
	m.launched = false
}

func (m *Module) Close() {
	if !m.connected {
		return
	}
	m.instance.connect.Close()
	m.connected = false
	m.initialized = false
}
