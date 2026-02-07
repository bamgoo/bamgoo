package bus

import "github.com/bamgoo/bamgoo"

var (
	host bamgoo.Host
)

func init() {
	host = bamgoo.Mount(module)
}
