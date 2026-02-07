package config

import (
	"github.com/bamgoo/bamgoo/base"
)

type (
	Driver interface {
		Load(base.Map) (base.Map, error)
	}
)
