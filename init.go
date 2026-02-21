package bamgoo

func init() {
	Mount(core)
	Mount(basic)
	Mount(library)
	Mount(trigger)
	Mount(providers)

	hook.AttachBus(&defaultBusHook{})
	hook.AttachConfig(&defaultConfigHook{})
}
