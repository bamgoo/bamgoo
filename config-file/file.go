package config_file

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/pelletier/go-toml/v2"

	"github.com/bamgoo/bamgoo"
	. "github.com/bamgoo/bamgoo/base"
)

type FileConfigDriver struct{}

func init() {
	bamgoo.Register("file", &FileConfigDriver{})
}

func (d *FileConfigDriver) Load(params Map) (Map, error) {
	file := "config.toml"
	if vv, ok := params["file"].(string); ok {
		file = vv
	}
	if vv, ok := params["path"].(string); ok {
		file = vv
	}
	if vv, ok := params["config"].(string); ok {
		file = vv
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
		format = detectFormat(data)
	}

	return decodeConfig(data, format)
}

func decodeConfig(data []byte, format string) (Map, error) {
	var out Map
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

func detectFormat(data []byte) string {
	s := strings.TrimSpace(string(data))
	if strings.HasPrefix(s, "{") || strings.HasPrefix(s, "[") {
		return "json"
	}
	return "toml"
}
