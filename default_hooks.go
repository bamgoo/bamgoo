package bamgoo

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"time"

	base "github.com/bamgoo/base"
	"github.com/pelletier/go-toml/v2"
)

type defaultBusHook struct{}

type defaultConfigHook struct{}

func (h *defaultBusHook) Request(meta *Meta, name string, value base.Map, _ time.Duration) (base.Map, base.Res) {
	data, res, ok := core.invokeLocal(meta, name, value)
	if ok {
		return data, res
	}
	return nil, OK
}

func (h *defaultBusHook) Publish(meta *Meta, name string, value base.Map) error {
	_, _, _ = core.invokeLocal(meta, name, value)
	return nil
}

func (h *defaultBusHook) Enqueue(meta *Meta, name string, value base.Map) error {
	go core.invokeLocal(meta, name, value)
	return nil
}

func (h *defaultBusHook) Stats() []ServiceStats {
	return nil
}

func (h *defaultConfigHook) LoadConfig() (base.Map, error) {
	drvName, params, err := parseConfigParams()
	if err != nil {
		return nil, err
	}
	if drvName == "" {
		return nil, nil
	}
	if drvName != DEFAULT && drvName != "file" {
		return nil, errors.New("Unknown config driver: " + drvName)
	}
	return loadConfigFromFile(params)
}

func parseConfigParams() (string, base.Map, error) {
	params := base.Map{}
	for k, v := range parseConfigEnv() {
		params[k] = v
	}
	for k, v := range parseConfigArgs() {
		params[k] = v
	}

	driver := DEFAULT
	if v, ok := params["driver"].(string); ok && v != "" {
		driver = v
	}
	if driver == "" {
		driver = DEFAULT
	}
	if driver == DEFAULT || driver == "file" {
		if _, ok := params["file"]; !ok {
			if file := defaultConfigFile(); file != "" {
				params["file"] = file
			}
		}
	}
	return driver, params, nil
}

func parseConfigEnv() base.Map {
	envs := os.Environ()
	params := base.Map{}
	for _, kv := range envs {
		parts := strings.SplitN(kv, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := parts[0]
		val := parts[1]
		if !strings.HasPrefix(key, "BAMGOO_") {
			continue
		}
		k := strings.ToLower(strings.TrimPrefix(key, "BAMGOO_"))
		params[k] = val
	}
	return params
}

func parseConfigArgs() base.Map {
	args := os.Args[1:]
	params := base.Map{}

	if len(args) == 1 {
		params["driver"] = DEFAULT
		params["file"] = args[0]
		return params
	}

	for i := 0; i < len(args); i++ {
		arg := args[i]
		if !strings.HasPrefix(arg, "--") {
			if i == 0 {
				params["driver"] = arg
			}
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

func loadConfigFromFile(params base.Map) (base.Map, error) {
	file := ""
	if vv, ok := params["file"].(string); ok {
		file = vv
	}
	if vv, ok := params["path"].(string); ok {
		file = vv
	}
	if vv, ok := params["config"].(string); ok {
		file = vv
	}
	if file == "" {
		file = defaultConfigFile()
	}
	if file == "" {
		return nil, nil
	}

	data, err := os.ReadFile(file)
	if err != nil {
		return nil, err
	}
	format, _ := params["format"].(string)
	if format == "" {
		ext := strings.ToLower(filepath.Ext(file))
		switch ext {
		case ".json":
			format = "json"
		case ".toml", ".tml":
			format = "toml"
		}
	}
	if format == "" {
		format = detectConfigFormat(data)
	}
	return decodeConfig(data, format)
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

func detectConfigFormat(data []byte) string {
	str := strings.TrimSpace(string(data))
	if strings.HasPrefix(str, "{") || strings.HasPrefix(str, "[") {
		return "json"
	}
	if str != "" {
		return "toml"
	}
	return ""
}

func decodeConfig(data []byte, format string) (base.Map, error) {
	var out base.Map
	switch strings.ToLower(format) {
	case "json":
		if err := json.Unmarshal(data, &out); err != nil {
			return nil, err
		}
		return out, nil
	case "toml":
		if err := toml.Unmarshal(data, &out); err != nil {
			return nil, err
		}
		return out, nil
	default:
		return nil, errors.New("Unknown config format: " + format)
	}
}
