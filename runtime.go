package bamgoo

import (
	"os"
	"os/signal"
	"slices"
	"sync"
	"syscall"

	. "github.com/bamgoo/bamgoo/base"
)

// bamgoo is the bamgoo runtime instance that drives module lifecycle.
var bamgoo = &bamgooRuntime{
	modules: make([]Module, 0),
	name:    BAMGOO, role: BAMGOO, node: "", version: "", setting: Map{},
}

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

type bamgooRuntime struct {
	mutex   sync.RWMutex
	modules []Module

	name    string
	role    string
	node    string
	version string
	setting Map

	overrideStatus bool
	configStatus   bool
	setupStatus    bool
	openStatus     bool
	startStatus    bool
	closeStatus    bool
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
func (c *bamgooRuntime) runtimeConfig(cfg Map) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if c.setupStatus || c.openStatus || c.startStatus {
		return
	}

	if cfg == nil {
		cfg = Map{}
	}

	if name, ok := cfg["name"].(string); ok && name != "" {
		c.name = name
	}
	if role, ok := cfg["role"].(string); ok {
		c.role = role
	}
	if node, ok := cfg["node"].(string); ok && node != "" {
		c.node = node
	}
	if version, ok := cfg["version"].(string); ok {
		c.version = version
	}
	if setting, ok := cfg["setting"].(Map); ok {
		for k, v := range setting {
			c.setting[k] = v
		}
	}

	c.configStatus = true
}

// Config applies config to core and all modules.
func (c *bamgooRuntime) Config(cfg Map) {
	if cfg == nil {
		cfg = Map{}
	}

	c.runtimeConfig(cfg)
	for _, mod := range c.modules {
		mod.Config(cfg)
	}
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
	// close the modules in reverse order
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
