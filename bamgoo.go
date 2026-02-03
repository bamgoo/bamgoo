package bamgoo

import (
	"os"
	"os/signal"
	"slices"
	"sync"
	"syscall"

	. "github.com/bamgoo/bamgoo/base"
)

const (
	BAMGOO = "bamgoo"
)

type (
	Module interface {
		Register(string, Any)
		Config(Map)
		Setup()
		Open()
		Start()
		Stop()
		Close()
	}
)

// bamgoo is the bamgoo runtime instance that drives module lifecycle.
var bamgoo = &bamgooRuntime{
	config: bamgooConfig{
		name: BAMGOO, role: BAMGOO, node: "", version: "",
		secret: BAMGOO, salt: BAMGOO,
	},
	setting: Map{},
	modules: make([]Module, 0),
}

type bamgooRuntime struct {
	mutex   sync.RWMutex
	modules []Module
	config  bamgooConfig
	setting Map

	overrideStatus bool
	configStatus   bool
	setupStatus    bool
	openStatus     bool
	startStatus    bool
	closeStatus    bool
}

type bamgooConfig struct {
	name    string
	role    string
	node    string
	version string
	secret  string
	salt    string
	setting Map
}

// Mount attaches a module into the core lifecycle.
func (c *bamgooRuntime) Mount(mod Module) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	// check if the module is already mounted
	if slices.Contains(c.modules, mod) {
		return
	}

	// append the module to the modules list
	c.modules = append(c.modules, mod)
}

// Register dispatches registrations to all mounted modules.
func (c *bamgooRuntime) Register(name string, value Any) {
	// if the value is a module, mount it
	if mod, ok := value.(Module); ok {
		c.Mount(mod)
		return
	}

	// if the value is a config, update the config
	if cfg, ok := value.(Map); ok {
		c.Config(cfg)
	}

	// dispatch the registration to all mounted modules
	for _, mod := range c.modules {
		mod.Register(name, value)
	}
}

// Config updates core config and broadcasts to modules.
func (c *bamgooRuntime) Config(cfg Map) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if c.setupStatus || c.openStatus || c.startStatus {
		return
	}

	if cfg == nil {
		cfg = Map{}
	}

	if name, ok := cfg["name"].(string); ok && name != "" {
		c.config.name = name
		if c.config.secret == "" {
			c.config.secret = name
		}
	}
	if role, ok := cfg["role"].(string); ok {
		c.config.role = role
	}
	if node, ok := cfg["node"].(string); ok && node != "" {
		c.config.node = node
	}
	if version, ok := cfg["version"].(string); ok {
		c.config.version = version
	}
	if secret, ok := cfg["secret"].(string); ok && secret != "" {
		c.config.secret = secret
	}
	if salt, ok := cfg["salt"].(string); ok && salt != "" {
		c.config.salt = salt
	}
	if setting, ok := cfg["setting"].(Map); ok {
		for k, v := range setting {
			c.config.setting[k] = v
		}
	}

	c.configStatus = true
}

// Setup initializes all modules.
func (c *bamgooRuntime) Setup() {
	if c.setupStatus {
		return
	}
	for _, mod := range c.modules {
		mod.Setup()
	}
	c.setupStatus = true
	c.closeStatus = false
}

// Open connects all modules.
func (c *bamgooRuntime) Open() {
	if c.openStatus {
		return
	}
	for _, mod := range c.modules {
		mod.Open()
	}
	c.openStatus = true
}

// Start launches all modules.
func (c *bamgooRuntime) Start() {
	if c.startStatus {
		return
	}
	for _, mod := range c.modules {
		mod.Start()
	}
	c.startStatus = true
}

// Stop terminates all modules in reverse order.
func (c *bamgooRuntime) Stop() {
	if !c.startStatus {
		return
	}
	// stop the modules in reverse order
	for i := len(c.modules) - 1; i >= 0; i-- {
		c.modules[i].Stop()
	}
	c.startStatus = false
}

// Close releases resources for all modules in reverse order.
func (c *bamgooRuntime) Close() {
	if c.closeStatus {
		return
	}
	for i := len(c.modules) - 1; i >= 0; i-- {
		c.modules[i].Close()
	}
	c.closeStatus = true
	c.openStatus = false
	c.setupStatus = false
}

// Wait blocks until system termination signal.
func (c *bamgooRuntime) Wait() {
	waiter := make(chan os.Signal, 1)
	signal.Notify(waiter, os.Interrupt, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	<-waiter
}

// Override controls whether registrations can overwrite existing entries.
func (c *bamgooRuntime) Override(args ...bool) bool {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	if len(args) > 0 {
		c.overrideStatus = args[0]
	}
	return c.overrideStatus
}
