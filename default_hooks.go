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
	file := configFileFromEnv()
	if file == "" {
		file = configFileFromArgs(os.Args[1:])
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
	format := detectConfigFormat(file, data)
	if format == "" {
		return nil, errors.New("Unknown config format")
	}
	return decodeConfig(data, format)
}

func configFileFromEnv() string {
	if v := os.Getenv("BAMGOO_CONFIG_FILE"); v != "" {
		return v
	}
	if v := os.Getenv("BAMGOO_CONFIG_PATH"); v != "" {
		return v
	}
	if v := os.Getenv("BAMGOO_CONFIG"); v != "" {
		return v
	}
	return ""
}

func configFileFromArgs(args []string) string {
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if !strings.HasPrefix(arg, "--") {
			continue
		}
		kv := strings.TrimPrefix(arg, "--")
		if kv == "" {
			continue
		}
		key := ""
		val := ""
		if strings.Contains(kv, "=") {
			parts := strings.SplitN(kv, "=", 2)
			key = strings.ToLower(parts[0])
			val = parts[1]
		} else if i+1 < len(args) && !strings.HasPrefix(args[i+1], "--") {
			key = strings.ToLower(kv)
			val = args[i+1]
			i++
		}
		if key == "config" || key == "config_file" || key == "config_path" || key == "file" || key == "path" {
			if val != "" {
				return val
			}
		}
	}
	return ""
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

func detectConfigFormat(file string, data []byte) string {
	if ext := strings.ToLower(filepath.Ext(file)); ext != "" {
		switch ext {
		case ".json":
			return "json"
		case ".toml", ".tml":
			return "toml"
		}
	}
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
