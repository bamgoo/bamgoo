package builtin

import (
	_ "github.com/bamgoo/bamgoo/config"
	_ "github.com/bamgoo/bamgoo/config-file"
	_ "github.com/bamgoo/bamgoo/config-redis"

	_ "github.com/bamgoo/bamgoo/bus"
	_ "github.com/bamgoo/bamgoo/bus-default"
	_ "github.com/bamgoo/bamgoo/bus-nats"
)
