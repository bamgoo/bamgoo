package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/bamgoo/bamgoo"
	. "github.com/bamgoo/bamgoo/base"
)

const (
	configEnvPrefix = "BAMGOO_"
)

var (
	errConfigDriverNotFound = errors.New("config driver not found")
	errConfigSourceNotFound = errors.New("config source not found")
)

var (
	module = &Module{drivers: map[string]Driver{}}
	host   = bamgoo.Mount(module)
)

type (
	Module struct {
		drivers map[string]Driver
	}
	Driver interface {
		Load(Map) (Map, error)
	}
)

// Register dispatches config driver registrations.
func (c *Module) Register(name string, value Any) {
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
func (c *Module) Config(Map) {}
func (c *Module) Setup()     {}
func (c *Module) Open()      {}
func (c *Module) Start() {
	fmt.Println("config module is running.")
}
func (c *Module) Stop()  {}
func (c *Module) Close() {}

func (c *Module) LoadConfig() (Map, error) {
	params, driverName, ok, err := c.Parse()
	if err != nil {
		return nil, err
	}

	fmt.Println("LoadConfig", ok, driverName, params)

	if !ok {
		return nil, errConfigSourceNotFound
	}
	if driverName == "" {
		return nil, errConfigSourceNotFound
	}

	driver, ok := c.drivers[driverName]
	fmt.Println("ddd", ok, c.drivers)
	if !ok {
		return nil, errors.New("Unknown config driver: " + driverName)
	}
	cfg, err := driver.Load(params)
	fmt.Println("load", err, cfg)
	return cfg, err
}

// Parse reads env (BAMGOO_*) then args (--key) and returns params + driver name.
func (c *Module) Parse() (Map, string, bool, error) {
	params := Map{}

	// env first
	for k, v := range c.parseEnv() {
		params[k] = v
	}
	// args override env
	for k, v := range c.parseArgs() {
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

		driver = "file"

		params["file"] = file
		params["path"] = file
		params["config"] = file
	}

	return params, driver, true, nil
}

func (c *Module) parseEnv() Map {
	envs := os.Environ()
	params := Map{}

	for _, kv := range envs {
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

func (c *Module) parseArgs() Map {
	args := os.Args[1:]
	params := Map{}

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
