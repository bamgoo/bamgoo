package config_redis

import "strings"

func detectFormat(data []byte) string {
	s := strings.TrimSpace(string(data))
	if strings.HasPrefix(s, "{") || strings.HasPrefix(s, "[") {
		return "json"
	}
	return "toml"
}
