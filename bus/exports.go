package bus

import "github.com/bamgoo/bamgoo"

// Register is a convenience wrapper so users can do:
// bamgoo.Register("file", bus.Driver)
// or bamgoo.Register("default", bus.BusConfig{...})
func Register(args ...any) {
	bamgoo.Register(args...)
}
