package bus_nats

import (
	"github.com/bamgoo/bamgoo"
)

// Driver returns the NATS bus driver.
func Driver() bamgoo.Driver {
	return &natsDriver{}
}

func init() {
	bamgoo.Register("nats", Driver())
}
