package main

import (
	"github.com/bamgoo/bamgoo"
	. "github.com/bamgoo/bamgoo/base"
)

func main() {
	bamgoo.Go()
}

func init() {

	bamgoo.Register("nats", bamgoo.BusConfig{
		Driver: "nats",
	})

	bamgoo.Register("test.get", bamgoo.Service{
		Name: "查询", Desc: "查询",
		Action: func(ctx *bamgoo.Context) (Map, Res) {
			return Map{"msg": "get from node 1"}, nil
		},
	})

}
