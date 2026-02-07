package main

import (
	"github.com/bamgoo/bamgoo"
	. "github.com/bamgoo/bamgoo/base"
	"github.com/bamgoo/bamgoo/bus"
	_ "github.com/bamgoo/bamgoo/bus"
	_ "github.com/bamgoo/bamgoo/bus-default"
	_ "github.com/bamgoo/bamgoo/bus-nats"
	_ "github.com/bamgoo/bamgoo/config"
	_ "github.com/bamgoo/bamgoo/config-file"
	_ "github.com/bamgoo/bamgoo/config-redis"
)

func main() {
	bamgoo.Go()
}

func init() {

	bamgoo.Register("nats", bus.BusConfig{
		Driver: "nats",
	})

	bamgoo.Register("test.get", bamgoo.Service{
		Name: "查询", Desc: "查询",
		Action: func(ctx *bamgoo.Context) (Map, Res) {
			return Map{"msg": "fail from node 3"}, bamgoo.Fail
		},
	})

}
