package config

import (
	"github.com/bamgoo/bamgoo"
)

var (
	host = bamgoo.Mount(module)
)

func init() {
	// host.InvokeLocal(nil, "test", Map{"asd": 123})
}
