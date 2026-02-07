package bamgoo

import (
	. "github.com/bamgoo/bamgoo/base"
)

// Mount attaches a module into the bamgoo runtime and returns a host.
func Mount(mod Module) Host {
	return bamgoo.Mount(mod)
}

// Register registers anything into mounted modules.
func Register(args ...Any) {
	name := ""
	values := make([]Any, 0)
	for _, arg := range args {
		switch v := arg.(type) {
		case string:
			name = v
		default:
			values = append(values, v)
		}
	}

	for _, value := range values {
		bamgoo.Register(name, value)
	}
}

// Ready initializes and connects modules without starting them.
func Ready() {
	bamgoo.Setup()
	bamgoo.Open()
}

// Go starts the full lifecycle and blocks until stop.
func Go() {
	bamgoo.Setup()
	bamgoo.Open()
	bamgoo.Start()
	bamgoo.Wait()
	bamgoo.Stop()
	bamgoo.Close()
}

// Override controls whether registrations can overwrite existing entries.
func Override(args ...bool) bool {
	return bamgoo.Override(args...)
}
