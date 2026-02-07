package config

import (
	base "github.com/bamgoo/bamgoo/base"
)

type (
	Driver interface {
		Load(base.Map) (base.Map, error)
	}
)
