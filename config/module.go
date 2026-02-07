package config

import (
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/bamgoo/bamgoo"
	base "github.com/bamgoo/bamgoo/base"
)

const (
	configEnvPrefix = "BAMGOO_"
)

var (
	errConfigDriverNotFound = errors.New("config driver not found")
	errConfigSourceNotFound = errors.New("config source not found")
)

var module = &Module{drivers: map[string]Driver{}}

type (
	Module struct {
		drivers map[string]Driver
	}
)

func init() {
	bamgoo.Register(module) // register as module if needed
}

// Register dispatches config driver registrations.
func (c *Module) Register(name string, value base.Any) {
	if drv, ok := value.(Driver); ok {
		c.RegisterDriver(name, drv)
	}
}

func (c *Module) RegisterDriver(name string, driver Driver) {
	if name == "" {
		name = bamgoo.DEFAULT
	}
	if driver == nil {
		panic("Invalid config driver: " + name)
	}
	if _, ok := c.drivers[name]; ok {
		panic("Config driver already registered: " + name)
	}
	c.drivers[name] = driver
}

// Module methods (no-op for now)
func (c *Module) Config(base.Map) {}
func (c *Module) Setup()          {}
func (c *Module) Open()           {}
func (c *Module) Start()          {}
func (c *Module) Stop()           {}
func (c *Module) Close()          {}

// Parse reads env (BAMGOO_*) then args (--key) and returns params + driver name.
func (c *Module) Parse(env []string, args []string) (base.Map, string, bool, error) {
	params := base.Map{}

	// env first
	for k, v := range c.parseEnv(env) {
		params[k] = v
	}
	// args override env
	for k, v := range c.parseArgs(args) {
		params[k] = v
	}

	driver := ""
	if v, ok := params["config_driver"].(string); ok && v != "" {
		driver = v
	}
	if driver == "" {
		if v, ok := params["driver"].(string); ok && v != "" {
			driver = v
		}
	}

	if driver == "" {
		file := defaultConfigFile()
		if file == "" {
			return nil, "", false, nil
		}
		params["file"] = file
		driver = "file"
	}

	return params, driver, true, nil
}

func (c *Module) LoadConfig() (base.Map, error) {
	params, driverName, ok, err := c.Parse(os.Environ(), os.Args[1:])
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, errConfigSourceNotFound
	}
	if driverName == "" {
		return nil, errConfigSourceNotFound
	}

	driver, ok := c.drivers[driverName]
	if !ok {
		return nil, errors.New("Unknown config driver: " + driverName)
	}
	return driver.Load(params)
}

func (c *Module) parseEnv(env []string) base.Map {
	params := base.Map{}
	for _, kv := range env {
		parts := strings.SplitN(kv, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := parts[0]
		val := parts[1]
		if !strings.HasPrefix(key, configEnvPrefix) {
			continue
		}
		k := strings.ToLower(strings.TrimPrefix(key, configEnvPrefix))
		params[k] = val
	}
	return params
}

func (c *Module) parseArgs(args []string) base.Map {
	params := base.Map{}
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if !strings.HasPrefix(arg, "--") {
			continue
		}
		kv := strings.TrimPrefix(arg, "--")
		if kv == "" {
			continue
		}
		if strings.Contains(kv, "=") {
			parts := strings.SplitN(kv, "=", 2)
			params[strings.ToLower(parts[0])] = parts[1]
			continue
		}
		if i+1 < len(args) && !strings.HasPrefix(args[i+1], "--") {
			params[strings.ToLower(kv)] = args[i+1]
			i++
		} else {
			params[strings.ToLower(kv)] = "true"
		}
	}
	return params
}

func defaultConfigFile() string {
	candidates := []string{"config.toml", "config.json"}

	if exe := filepath.Base(os.Args[0]); exe != "" {
		name := strings.TrimSuffix(exe, filepath.Ext(exe))
		candidates = append(candidates, name+".toml", name+".json")
	}

	for _, file := range candidates {
		if _, err := os.Stat(file); err == nil {
			return file
		}
	}
	return ""
}
