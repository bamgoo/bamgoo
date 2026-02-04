package bamgoo

// init registers all internal modules and drivers in the correct order.
// Order matters:
// 1. core    - foundation module for method/service registration
// 2. bus     - message bus module
// 3. trigger - trigger/hook module
// 4. drivers - must be registered after their modules are mounted
func init() {
	// 1. Mount core modules in order
	Mount(core)
	Mount(bus)
	Mount(trigger)

	// 2. Register bus drivers
	Register(DEFAULT, &defaultBusDriver{})
	Register("nats", &natsBusDriver{})
}
