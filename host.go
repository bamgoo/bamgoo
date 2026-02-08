package bamgoo

import (
	. "github.com/bamgoo/base"
)

var host = &bamgooHost{}

type (
	bamgooHost struct {
	}

	Host interface {
		InvokeLocal(meta *Meta, name string, value Map) (Map, Res, bool)
	}
)

func (h *bamgooHost) InvokeLocal(meta *Meta, name string, value Map) (Map, Res, bool) {
	return core.invokeLocal(meta, name, value)
}
