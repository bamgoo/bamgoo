package bamgoo

func init() {
	Mount(core)
	Mount(basic)
	Mount(trigger)

	hook.AttachBus(&defaultBusHook{})
	hook.AttachConfig(&defaultConfigHook{})
}
