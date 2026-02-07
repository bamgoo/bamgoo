package config_file

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/bamgoo/bamgoo"
	base "github.com/bamgoo/bamgoo/base"
	"github.com/pelletier/go-toml/v2"
)

type fileConfigDriver struct{}

func init() {
	bamgoo.Register("file", &fileConfigDriver{})
}

func (d *fileConfigDriver) Load(params base.Map) (base.Map, error) {
	file, _ := params["file"].(string)
	if file == "" {
		file, _ = params["path"].(string)
	}
	if file == "" {
		return nil, errors.New("Missing config file")
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
